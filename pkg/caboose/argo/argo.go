package argo

import (
	"context"
	"fmt"
	"log"
	"os"
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

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	commonutil "github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/element84/swoop-go/pkg/caboose"
	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
	"github.com/element84/swoop-go/pkg/states"
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

func indexFn(obj any) ([]string, error) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, nil
	}

	phase, ok := un.GetLabels()[common.LabelKeyPhase]
	if !ok {
		// default phase to pending
		phase = string(v1alpha1.NodePending)
	}
	return []string{phase}, nil
}

func statusFromPhase(phase string) (states.WorkflowState, error) {
	if phase == "Succeeded" {
		return states.WorkflowState(states.Successful), nil
	} else if phase == "Error" {
		return states.WorkflowState(states.Failed), nil
	}

	return states.ParseWorkflowState(phase)
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
	properties *caboose.WorkflowProperties
}

type argoCabooseRunner struct {
	s3          *s3.SwoopS3
	callbackMap caboose.CallbackMap
	ctx         context.Context
	db          *pgxpool.Pool
	wfClientSet wfclientset.Interface
	dynIface    *dynamic.DynamicClient
	wg          *sync.WaitGroup
	wfChan      chan *workflowEvent
}

func (acr *argoCabooseRunner) newWorkflowEvent(eventType wfEventType, raw any) (*workflowEvent, error) {
	properties, err := acr.newWorkflowProperties(raw)
	if err != nil {
		return nil, err
	}

	return &workflowEvent{
		eventType:  eventType,
		wf:         raw,
		properties: properties,
	}, nil
}

func (acr *argoCabooseRunner) newWorkflowProperties(raw any) (*caboose.WorkflowProperties, error) {
	un, ok := raw.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to parse workflow: %v", raw)
	}

	labels := un.GetLabels()
	statusMap, _, _ := unstructured.NestedMap(un.Object, "status")
	start, _, _ := unstructured.NestedString(statusMap, "startedAt")
	finish, _, _ := unstructured.NestedString(statusMap, "finishedAt")

	p := &caboose.WorkflowProperties{
		Uuid: uuid.FromStringOrNil(un.GetName()),
	}

	phase := labels[common.LabelKeyPhase]
	status, err := statusFromPhase(phase)
	if err != nil {
		return nil, fmt.Errorf(
			"Cannot map workflow phase '%s' to known status; cannot process workflow %v",
			phase,
			raw,
		)
	}
	p.Status = status

	p.StartedAt, _ = time.Parse(time.RFC3339, start)
	p.FinishedAt, _ = time.Parse(time.RFC3339, finish)
	p.ErrorMsg, _, _ = unstructured.NestedString(statusMap, "message")

	if p.Uuid.IsNil() {
		return nil, fmt.Errorf("unknown workflow: %v", raw)
	}

	// TODO: need a label on workflows to prevent this lookup
	err = p.LookupName(acr.ctx, acr.db)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to lookup workflow name for uuid '%s': %s",
			p.Uuid,
			err,
		)
	}

	return p, nil
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
				wf.properties.Uuid,
				err,
			)
			go acr.backoff(wf)
		}
	}
}

func (acr *argoCabooseRunner) backoff(wf *workflowEvent) {
	backoffSecs := time.Duration(utils.IntPow(2, wf.retries)) * minBackoff
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
	err := wf.properties.ToStartEvent().Insert(acr.ctx, acr.db)
	if err != nil {
		return err
	}
	log.Printf(
		"Inserted start event for workflow: '%s'",
		wf.properties.Uuid,
	)
	return nil
}

func (acr *argoCabooseRunner) wfDone(wf *workflowEvent) error {
	tx, err := acr.db.Begin(acr.ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(acr.ctx)

	err = wf.properties.ToStartEvent().Insert(acr.ctx, acr.db)
	if err != nil {
		return err
	}
	log.Printf(
		"Inserted start event for workflow: '%s'",
		wf.properties.Uuid,
	)

	err = wf.properties.ToEndEvent().Insert(acr.ctx, acr.db)
	if err != nil {
		return err
	}
	log.Printf(
		"Inserted end event for workflow: '%s'",
		wf.properties.Uuid,
	)

	err = acr.s3.PutWorkflowResource(acr.ctx, wf.properties.Uuid, wf.wf)
	if err != nil {
		return err
	}

	callbacks, ok := acr.callbackMap.Lookup(
		wf.properties.Name,
		states.FinalState(wf.properties.Status),
	)
	if !ok {
		log.Printf(
			"No callbacks found for workflow '%s' with status '%s'",
			wf.properties.Name,
			wf.properties.Status,
		)
	}

	err = caboose.NewCallbackExecutor(
		acr.ctx,
		acr.s3,
		acr.db,
	).ProcessCallbacks(callbacks, wf.properties)
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

func (acr *argoCabooseRunner) addWorkflowInformerHandlers(
	wfInformer cache.SharedIndexInformer,
) {
	handle := func(eventType wfEventType) func(interface{}) {
		return func(obj interface{}) {
			wf, err := acr.newWorkflowEvent(eventType, obj)
			if err != nil {
				log.Println(err)
				return
			}
			acr.wfChan <- wf
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

type ArgoCaboose struct {
	S3Driver       *s3.S3Driver
	SwoopConfig    *config.SwoopConfig
	K8sConfigFlags *genericclioptions.ConfigFlags
	DbConfig       *db.PoolConfig
}

func (c *ArgoCaboose) newArgoCabooseRunner(ctx context.Context) (*argoCabooseRunner, error) {
	// check connection to object storage
	// allows us to fail fast if creds are obviously bad,
	// but doesn't validate if we can actually write
	err := c.S3Driver.CheckConnect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed checking connnection to object storage: %s", err)
	}

	db, err := c.DbConfig.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %s", err)
	}

	restConfig, err := c.K8sConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %s", err)
	}

	wfClientSet := wfclientset.NewForConfigOrDie(restConfig)
	dynamicInterface := dynamic.NewForConfigOrDie(restConfig)

	wfChan := make(chan *workflowEvent)
	var wg sync.WaitGroup

	return &argoCabooseRunner{
		s3:          s3.NewSwoopS3(s3.NewJsonClient(c.S3Driver)),
		callbackMap: caboose.MapConfigCallbacks(c.SwoopConfig),
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

	wfInformer := util.NewWorkflowInformer(
		acr.dynIface,
		namespace,
		workflowResyncPeriod,
		func(options *metav1.ListOptions) {
			labelSelector := labels.NewSelector().
				Add(util.InstanceIDRequirement(instanceId))
			options.LabelSelector = labelSelector.String()
		},
		cache.Indexers{
			"workflow.phase": indexFn,
		},
	)

	acr.addWorkflowInformerHandlers(wfInformer)
	go wfInformer.Run(ctx.Done())

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
