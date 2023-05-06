package caboose

import (
	"context"
	"fmt"
	"log"
	"os"
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
	"github.com/vingarcia/ksql"

	"github.com/element84/swoop-go/pkg/db"
)

const (
	workflowResyncPeriod = 20 * time.Minute
	// TODO: make this a config parameter
	maxWorkers = 4
)

type wfEventType int

const (
	started   wfEventType = 0
	completed wfEventType = 1
)

type workflowEvent struct {
	eventType wfEventType
	wf        interface{}
}

func (wf *workflowEvent) process(acr *argoCabooseRunner) {
	switch wf.eventType {
	case started:
		acr.wfStart(wf.wf)
	case completed:
		acr.wfDone(wf.wf)
	}
}

type argoCabooseRunner struct {
	configFile string
	ctx        context.Context
	db         *ksql.DB
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

func (acr *argoCabooseRunner) wfStart(wf interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(wf)
	if err == nil {
		log.Printf("Recevied started wf key: %s", key)
		log.Printf("Obj: %v", wf.(*unstructured.Unstructured))
	}
}

func (acr *argoCabooseRunner) wfDone(wf interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(wf)
	if err == nil {
		log.Printf("Recevied completed wf key: %s", key)
		log.Printf("Obj: %v", wf.(*unstructured.Unstructured))
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
