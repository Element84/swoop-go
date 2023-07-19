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

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	commonutil "github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/utils"
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

func (p *workflowProperties) runCallback(cb *config.Callback) bool {
	if utils.Contains(cb.On, p.status) && !utils.Contains(cb.NotOn, p.status) {
		return true
	} else {
		return false
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
		return nil, fmt.Errorf("failed to parse workflow: %v", wf.wf)
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
		return nil, fmt.Errorf("unknown workflow: %v", wf.wf)
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
	swoopConfig *config.SwoopConfig
	ctx         context.Context
	db          *pgxpool.Pool
	wfClientSet wfclientset.Interface
	dynIface    *dynamic.DynamicClient
	wg          *sync.WaitGroup
	wfChan      chan workflowEvent
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

// TODO: how to handle retries?
func (acr *argoCabooseRunner) wfStart(wf *workflowEvent) {
	parsed, err := wf.extractProps()
	if err != nil {
		log.Printf("%s", err)
		return
	}

	_, err = parsed.toStartEvent().insert(acr.ctx, acr.db)
	if err != nil {
		log.Printf("%s", err)
	}
	log.Printf("Inserted start event for workflow: '%s'", parsed.workflowUUID)
}

// TODO: how to handle retries?
func (acr *argoCabooseRunner) wfDone(wf *workflowEvent) {
	fmt.Println("swoopconfig")
	fmt.Println(acr.swoopConfig)
	parsed, err := wf.extractProps()
	if err != nil {
		log.Printf("%s", err)
		return
	}
	_, err = parsed.toStartEvent().insert(acr.ctx, acr.db)
	if err != nil {
		log.Printf("%s", err)
		return
	}
	log.Printf("Inserted start event for workflow: '%s'", parsed.workflowUUID)

	_, err = parsed.toEndEvent().insert(acr.ctx, acr.db)
	if err != nil {
		log.Printf("%s", err)
		return
	}
	log.Printf("Inserted end event for workflow: '%s'", parsed.workflowUUID)

	// TODO: do we have a possible race here? If we delete the workflow before
	// argo has finished with cleanup or other post-workflow tasks, will it
	// lose the state it needs to finish them? How can we know the workflow is
	// good to clean up?
	err = acr.deleteWorkflow(wf)
	if err != nil {
		log.Printf("%s", err)
	}
}

func (acr *argoCabooseRunner) deleteWorkflow(wf *workflowEvent) error {
	key, err := cache.MetaNamespaceKeyFunc(wf.wf)
	if err != nil {
		return err
	}

	namespace, name, _ := cache.SplitMetaNamespaceKey(key)

	err = acr.wfClientSet.ArgoprojV1alpha1().Workflows(namespace).Delete(
		acr.ctx,
		name,
		metav1.DeleteOptions{
			PropagationPolicy: commonutil.GetDeletePropagation(),
		},
	)
	if err != nil {
		if apierr.IsNotFound(err) {
			log.Printf("Workflow already deleted '%s'", key)
		} else {
			return err
		}
	} else {
		log.Printf("Successfully requested to delete workflow '%s'", key)
	}

	return nil
}

type ArgoCaboose struct {
	SwoopConfig    *config.SwoopConfig
	K8sConfigFlags *genericclioptions.ConfigFlags
}

func (c *ArgoCaboose) newArgoCabooseRunner(ctx context.Context) (*argoCabooseRunner, error) {
	// db connection
	db, err := db.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %s", err)
	}

	// kube client
	restConfig, err := c.K8sConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %s", err)
	}

	wfClientSet := wfclientset.NewForConfigOrDie(restConfig)
	dynamicInterface := dynamic.NewForConfigOrDie(restConfig)

	wfChan := make(chan workflowEvent)
	var wg sync.WaitGroup

	return &argoCabooseRunner{
		swoopConfig: c.SwoopConfig,
		ctx:         ctx,
		db:          db,
		wfClientSet: wfClientSet,
		dynIface:    dynamicInterface,
		wg:          &wg,
		wfChan:      wfChan,
	}, nil
}

func (c *ArgoCaboose) Run(ctx context.Context, cancel context.CancelFunc) error {
	acr, err := c.newArgoCabooseRunner(ctx)
	if err != nil {
		return err
	}

	acr.StartWorkers()

	namespace := ""
	if c.K8sConfigFlags.Namespace != nil {
		namespace = *c.K8sConfigFlags.Namespace
	}

	// init wf informer
	wfInformer := util.NewWorkflowInformer(
		acr.dynIface,
		namespace, // namespace name
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
		return fmt.Errorf("timed out waiting for cache to sync")
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
