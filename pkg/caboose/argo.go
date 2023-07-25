package caboose

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
	"github.com/element84/swoop-go/pkg/utils"
)

const (
	// TODO: make these config parameters
	workflowResyncPeriod = 20 * time.Minute
	maxWorkers           = 4
	minBackoff           = 2 * time.Second
	maxBackoff           = 300 * time.Second
	instanceId           = ""
)

func IntPow(n, m int) int {
	if m == 0 {
		return 1
	}
	result := n
	for i := 2; i <= m; i++ {
		result *= n
	}
	return result
}

type workflowProperties struct {
	startedAt    time.Time
	finishedAt   time.Time
	workflowUUID uuid.UUID
	templateName string
	status       string
	errorMsg     string
}

func newWorkflowProperties(raw interface{}) (*workflowProperties, error) {
	un, ok := raw.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to parse workflow: %v", raw)
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
		return nil, fmt.Errorf("unknown workflow: %v", raw)
	}

	return p, nil
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

func (p *workflowProperties) toStartEvent() *db.Event {
	return &db.Event{
		ActionUUID: p.workflowUUID,
		EventTime:  p.startedAt,
		Status:     "RUNNING",
	}
}

func (p *workflowProperties) toEndEvent() *db.Event {
	return &db.Event{
		ActionUUID: p.workflowUUID,
		EventTime:  p.finishedAt,
		Status:     p.status,
		ErrorMsg:   p.errorMsg,
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
	eventType  wfEventType
	wf         interface{}
	retries    int
	properties *workflowProperties
}

func newWorkflowEvent(eventType wfEventType, raw interface{}) (*workflowEvent, error) {
	properties, err := newWorkflowProperties(raw)
	if err != nil {
		return nil, err
	}

	return &workflowEvent{
		eventType:  eventType,
		wf:         raw,
		properties: properties,
	}, nil
}

type argoCabooseRunner struct {
	s3Driver    *s3.S3Driver
	swoopConfig *config.SwoopConfig
	ctx         context.Context
	db          *pgxpool.Pool
	wfClientSet wfclientset.Interface
	dynIface    *dynamic.DynamicClient
	wg          *sync.WaitGroup
	wfChan      chan *workflowEvent
}

// TODO: how to restart failed workers?
//
//	Panic handler?
//	wait.Until?
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
			acr.process(wf)
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

func (acr *argoCabooseRunner) process(wf *workflowEvent) {
	switch wf.eventType {
	case started:
		acr.wfStart(wf)
	case completed:
		err := acr.wfDone(wf)
		if err != nil {
			log.Printf(
				"error encountered processing '%s': %s",
				wf.properties.workflowUUID,
				err,
			)
			go acr.backoff(wf)
		}
	}
}

func (acr *argoCabooseRunner) backoff(wf *workflowEvent) {
	backoffSecs := time.Duration(IntPow(2, wf.retries)) * minBackoff
	if backoffSecs > maxBackoff {
		backoffSecs = maxBackoff
	}

	select {
	case <-acr.ctx.Done():
		return
	case <-time.After(backoffSecs):
	}

	wf.retries++
	acr.wfChan <- wf
}

func (acr *argoCabooseRunner) wfStart(wf *workflowEvent) error {
	_, err := wf.properties.toStartEvent().Insert(acr.ctx, acr.db)
	if err != nil {
		return err
	}
	log.Printf(
		"Inserted start event for workflow: '%s'",
		wf.properties.workflowUUID,
	)
	return nil
}

func (acr *argoCabooseRunner) wfDone(wf *workflowEvent) error {
	tx, err := acr.db.Begin(acr.ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(acr.ctx)

	_, err = wf.properties.toStartEvent().Insert(acr.ctx, acr.db)
	if err != nil {
		return err
	}
	log.Printf(
		"Inserted start event for workflow: '%s'",
		wf.properties.workflowUUID,
	)

	_, err = wf.properties.toEndEvent().Insert(acr.ctx, acr.db)
	if err != nil {
		return err
	}
	log.Printf(
		"Inserted end event for workflow: '%s'",
		wf.properties.workflowUUID,
	)

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(wf.wf)
	err = acr.s3Driver.Put(
		acr.ctx,
		fmt.Sprintf(
			"executions/%s/workflow.json",
			wf.properties.workflowUUID,
		),
		b,
		int64(b.Len()),
	)
	if err != nil {
		return err
	}

	// TODO: do we have a possible race here? If we delete the workflow before
	// argo has finished with cleanup or other post-workflow tasks, will it
	// lose the state it needs to finish them? How can we know the workflow is
	// good to clean up?
	//
	// Maybe we apply a label, and use another listener to delete workflows
	// with said label (and perhaps another argo one)?
	//
	// Looks like argo is looking for `common.IsDone(un)`
	err = acr.deleteWorkflow(wf)
	if err != nil {
		return err
	}

	err = tx.Commit(acr.ctx)
	if err != nil {
		return err
	}

	return nil
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
	S3Driver       *s3.S3Driver
	SwoopConfig    *config.SwoopConfig
	K8sConfigFlags *genericclioptions.ConfigFlags
}

func (c *ArgoCaboose) newArgoCabooseRunner(ctx context.Context) (*argoCabooseRunner, error) {
	// check connection to object storage
	// allows us to fail fast if creds are obviously bad,
	// but doesn't validate if we can actually write
	err := c.S3Driver.CheckConnect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed checking connnection to object storage: %s", err)
	}

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

	wfChan := make(chan *workflowEvent)
	var wg sync.WaitGroup

	return &argoCabooseRunner{
		s3Driver:    c.S3Driver,
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
				Add(util.InstanceIDRequirement(instanceId))
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
	wfChan chan<- *workflowEvent,
) {
	handle := func(eventType wfEventType) func(interface{}) {
		return func(obj interface{}) {
			wf, err := newWorkflowEvent(eventType, obj)
			if err != nil {
				log.Println(err)
				return
			}
			wfChan <- wf
		}
	}

	// workflow start handler
	wfInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				un, ok := obj.(*unstructured.Unstructured)
				return ok && un.GetLabels()[common.LabelKeyPhase] == "Running"
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: handle(started),
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
				AddFunc: handle(completed),
			},
		},
	)
}
