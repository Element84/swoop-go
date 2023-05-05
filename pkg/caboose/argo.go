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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/util"

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

func (wf *workflowEvent) process() {
	switch wf.eventType {
	case started:
		wfStart(wf.wf)
	case completed:
		wfDone(wf.wf)
	}
}

type ArgoCaboose struct {
	ConfigFile     string
	DatabaseConfig *db.DatabaseConfig
	K8sConfigFlags *genericclioptions.ConfigFlags
}

func (c *ArgoCaboose) Run(ctx context.Context) error {
	log.Printf("Database URL: %s", c.DatabaseConfig.Url())

	restConfig, err := c.K8sConfigFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("Failed to get kubernetes config: %s", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("Could not create kubernetes client: %s", err)
	}

	vinfo, _ := k8sClientSet.Discovery().ServerVersion()
	log.Printf("K8s version: %s", vinfo)

	dynamicInterface, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// init wf informer
	wfInformer := util.NewWorkflowInformer(
		dynamicInterface,
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

	var wg sync.WaitGroup
	wfChan := make(chan workflowEvent)
	defer func() {
		close(wfChan)
	}()

	addWorkflowInformerHandlers(wfInformer, wfChan)
	go wfInformer.Run(ctx.Done())

	// wait until sync'd
	if !cache.WaitForCacheSync(
		ctx.Done(),
		wfInformer.HasSynced,
	) {
		return fmt.Errorf("Timed out waiting for cache to sync")
	}

	for i := 0; i < maxWorkers; i++ {
		go worker(i, ctx, &wg, wfChan)
	}

	<-ctx.Done()
	// TODO: what happens if a worker hangs? Workers should timeout?
	wg.Wait()
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

// TODO: look at errgroup to propagate errors back up (https://pkg.go.dev/golang.org/x/sync/errgroup)
func worker(id int, ctx context.Context, wg *sync.WaitGroup, wfChan <-chan workflowEvent) {
	log.Printf("starting worker %d", id)
	wg.Add(1)
	for {
		select {
		case wf := <-wfChan:
			wf.process()
		case <-ctx.Done():
			log.Printf("stopping worker %d", id)
			wg.Done()
			return
		}
	}
}

func wfStart(wf interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(wf)
	if err == nil {
		log.Printf("Recevied started wf key: %s", key)
		log.Printf("Obj: %v", wf.(*unstructured.Unstructured))
	}
}

func wfDone(wf interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(wf)
	if err == nil {
		log.Printf("Recevied completed wf key: %s", key)
		log.Printf("Obj: %v", wf.(*unstructured.Unstructured))
	}
}
