package caboose

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/element84/swoop-go/pkg/db"
)

const (
	workflowResyncPeriod = 20 * time.Minute
	// TODO: make this a config parameter
	maxWorkers = 4
)

type dbEvent struct {
	actionUUID uuid.UUID
	eventTime  time.Time
	status     string
	errorMsg   string
}

func (s *dbEvent) insert(ctx context.Context, db *pgxpool.Pool) (pgconn.CommandTag, error) {
	/*
		// We could do something like this if we wanted to prevent events being inserted for
		// unknown workflows. In reality, however, the current risk of not checking seems low.
		// If we did want this check, then it might make more sense as a trigger on event insert,
		// or perhaps a foreign key relation to action might be better (but runs into complications
		// with partitioning). For now we'll keep this here as a reference, in case we want it.
		var actionExists bool
		err := db.QueryRow(
			ctx,
			"SELECT exists(SELECT 1 from swoop.action where action_uuid = $1)",
			s.actionUUID,
		).Scan(&actionExists)

		if err != nil {
			// returning nil here doesn't work, we need a CommandTag
			return nil, err
		} else if !actionExists {
			// returning nil here doesn't work, we need a CommandTag
			return nil, fmt.Errorf("Cannot insert event, unknown action UUID: '%s'", s.actionUUID)
		}
	*/

	return db.Exec(
		ctx,
		`INSERT INTO swoop.event (
		    action_uuid,
			event_time,
			status,
			error,
			event_source
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			'swoop-caboose'
		) ON CONFLICT DO NOTHING`,
		s.actionUUID,
		s.eventTime,
		s.status,
		s.errorMsg,
	)
}

type workflowProperties struct {
	startedAt    time.Time
	finishedAt   time.Time
	workflowUUID uuid.UUID
	templateName string
	status       string
	errorMsg     string
}

func (p *workflowProperties) statusFromPhase(phase string) {
	if phase == "Succeeded" {
		p.status = "SUCCESSFUL"
	} else if phase == "Error" {
		p.status = "FAILED"
	} else {
		p.status = strings.ToUpper(phase)
	}
}

func (p *workflowProperties) toStartEvent() *dbEvent {
	return &dbEvent{
		actionUUID: p.workflowUUID,
		eventTime:  p.startedAt,
		status:     "RUNNING",
	}
}

func (p *workflowProperties) toEndEvent() *dbEvent {
	return &dbEvent{
		actionUUID: p.workflowUUID,
		eventTime:  p.finishedAt,
		status:     p.status,
		errorMsg:   p.errorMsg,
	}
}

type wfEventType int

const (
	started   wfEventType = 0
	completed wfEventType = 1
)

type workflowEvent struct {
	eventType wfEventType
	wf        interface{}
}

func (wf *workflowEvent) extractProps() (*workflowProperties, error) {
	un, ok := wf.wf.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("Failed to parse workflow: %v", wf.wf)
	}

	labels := un.GetLabels()
	status, _, _ := unstructured.NestedMap(un.Object, "status")
	start, _, _ := unstructured.NestedString(status, "startedAt")
	finish, _, _ := unstructured.NestedString(status, "finishedAt")

	p := &workflowProperties{
		workflowUUID: uuid.FromStringOrNil(un.GetName()),
		templateName: labels[common.LabelKeyWorkflowTemplate],
	}

	p.statusFromPhase(labels[common.LabelKeyPhase])
	p.startedAt, _ = time.Parse(time.RFC3339, start)
	p.finishedAt, _ = time.Parse(time.RFC3339, finish)
	p.errorMsg, _, _ = unstructured.NestedString(status, "message")

	if p.workflowUUID.IsNil() || p.templateName == "" {
		return nil, fmt.Errorf("Unknown workflow: %v", wf.wf)
	}

	return p, nil
}

func (wf *workflowEvent) process(acr *argoCabooseRunner) {
	switch wf.eventType {
	case started:
		acr.wfStart(wf)
	case completed:
		acr.wfDone(wf)
	}
}

type argoCabooseRunner struct {
	configFile string
	ctx        context.Context
	db         *pgxpool.Pool
	k8sClient  *dynamic.DynamicClient
	wg         *sync.WaitGroup
	wfChan     chan workflowEvent
}

// TODO: look at errgroup to propagate errors back up
//
//	(https://pkg.go.dev/golang.org/x/sync/errgroup)
func (acr *argoCabooseRunner) worker(id int) {
	log.Printf("starting worker %d", id)
	acr.wg.Add(1)
	defer acr.wg.Done()

	end := func() {
		log.Printf("stopping worker %d", id)
	}

	for {
		// first select is to give priority to ctx.Done
		select {
		case <-acr.ctx.Done():
			end()
			return
		default:
		}

		select {
		case wf := <-acr.wfChan:
			// TODO: what if the workflow isn't a
			// swoop one or is otherwise invalid?
			wf.process(acr)
		case <-acr.ctx.Done():
			end()
			return
		}

	}
}

func (acr *argoCabooseRunner) StartWorkers() {
	for i := 0; i < maxWorkers; i++ {
		go acr.worker(i)
	}
}

func (acr *argoCabooseRunner) wfStart(wf *workflowEvent) {
	parsed, err := wf.extractProps()
	if err != nil {
		log.Printf("%s", err)
		return
	}
	log.Printf("Started workflow props: %v", parsed)
	_, err = parsed.toStartEvent().insert(acr.ctx, acr.db)
	if err != nil {
		log.Printf("%s", err)
	}
}

func (acr *argoCabooseRunner) wfDone(wf *workflowEvent) {
	parsed, err := wf.extractProps()
	if err != nil {
		log.Printf("%s", err)
		return
	}
	log.Printf("Completed workflow props: %v", parsed)
	_, err = parsed.toStartEvent().insert(acr.ctx, acr.db)
	if err != nil {
		log.Printf("%s", err)
		return
	}
	_, err = parsed.toEndEvent().insert(acr.ctx, acr.db)
	if err != nil {
		log.Printf("%s", err)
	}
}

type ArgoCaboose struct {
	ConfigFile     string
	DatabaseConfig *db.DatabaseConfig
	K8sConfigFlags *genericclioptions.ConfigFlags
}

func (c *ArgoCaboose) newArgoCabooseRunner(ctx context.Context) (*argoCabooseRunner, error) {
	// db connection
	db, err := c.DatabaseConfig.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %s", err)
	}

	// kube client
	restConfig, err := c.K8sConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to get kubernetes config: %s", err)
	}

	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	wfChan := make(chan workflowEvent)
	var wg sync.WaitGroup

	return &argoCabooseRunner{
		configFile: c.ConfigFile,
		ctx:        ctx,
		db:         db,
		k8sClient:  client,
		wg:         &wg,
		wfChan:     wfChan,
	}, nil
}

func (c *ArgoCaboose) Run(ctx context.Context, cancel context.CancelFunc) error {
	acr, err := c.newArgoCabooseRunner(ctx)
	if err != nil {
		return err
	}

	acr.StartWorkers()

	// init wf informer
	wfInformer := util.NewWorkflowInformer(
		acr.k8sClient,
		// TODO: make namespace config parameter
		"argo-workflows", // namespace name
		workflowResyncPeriod,
		func(options *metav1.ListOptions) {
			labelSelector := labels.NewSelector().
				// TODO: make instanceID config parameter
				Add(util.InstanceIDRequirement("")) // instanceID
			options.LabelSelector = labelSelector.String()
		},
		cache.Indexers{
			indexes.WorkflowPhaseIndex: indexes.MetaWorkflowPhaseIndexFunc(),
		},
	)

	addWorkflowInformerHandlers(wfInformer, acr.wfChan)
	go wfInformer.Run(ctx.Done())

	// wait until sync'd
	if !cache.WaitForCacheSync(
		ctx.Done(),
		wfInformer.HasSynced,
	) {
		return fmt.Errorf("Timed out waiting for cache to sync")
	}

	// TODO: what happens if a worker hangs? Workers should timeout?
	// we wait on the workers, as they are waiting on ctx
	acr.wg.Wait()
	// if all the workers exit we cancel to make sure everything else stops
	// TODO may want to consider a solution like https://github.com/jackc/puddle
	//      to ensure dying workers are revived
	cancel()
	return nil
}

func (c *ArgoCaboose) SignalHandler(
	signalChan <-chan os.Signal,
	ctx context.Context,
	cancel context.CancelFunc,
) {
	select {
	case sig := <-signalChan:
		switch sig {
		case syscall.SIGINT:
			log.Printf("Got SIGINT, exiting.")
			cancel()
		case syscall.SIGTERM:
			log.Printf("Got SIGTERM, exiting.")
			cancel()
		}
	case <-ctx.Done():
		log.Printf("Done.")
	}
}

func addWorkflowInformerHandlers(
	wfInformer cache.SharedIndexInformer,
	wfChan chan<- workflowEvent,
) {
	// workflow start handler
	wfInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				un, ok := obj.(*unstructured.Unstructured)
				return ok && un.GetLabels()[common.LabelKeyPhase] == "Running"
				// TODO: use the extract function methods to filter based on valid wf or not
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					wfChan <- workflowEvent{started, obj}
				},
			},
		},
	)
	// workflow completion handler
	wfInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				un, ok := obj.(*unstructured.Unstructured)
				return ok && un.GetLabels()[common.LabelKeyCompleted] == "true"
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					wfChan <- workflowEvent{completed, obj}
				},
			},
		},
	)
}
