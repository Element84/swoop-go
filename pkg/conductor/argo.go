package conductor

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/gofrs/uuid/v5"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"

	"github.com/element84/swoop-go/pkg/config"
	sctx "github.com/element84/swoop-go/pkg/context"
	"github.com/element84/swoop-go/pkg/db"
)

type ArgoWorkflow struct {
	resourceKind        string
	resourceName        string
	submitOptsGenerator config.ArgoSubmitOptsGenerator
}

func ArgoWorkflowFromWorkflow(wf *config.Workflow) (*ArgoWorkflow, error) {
	if wf.ArgoOpts == nil {
		return nil, errors.New("cannot create ArgoWorkflow without Workflow.ArgoOpts defined")
	}

	sog, err := wf.ArgoOpts.SubmitOptsGenerator()
	if err != nil {
		return nil, err
	}

	awf := &ArgoWorkflow{
		resourceKind:        wf.ArgoOpts.Template.Kind,
		resourceName:        wf.ArgoOpts.Template.Name,
		submitOptsGenerator: sog,
	}

	return awf, nil
}

func (awf *ArgoWorkflow) SubmitWorkflow(
	ctx context.Context,
	ac *ArgoClient,
	wfUuid uuid.UUID,
	priority int,
) error {
	ctx, _ = sctx.MergeCancel(ac.ctx, ctx)

	_, err := ac.serviceClient.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
		Namespace:     ac.namespace,
		ResourceKind:  awf.resourceKind,
		ResourceName:  awf.resourceName,
		SubmitOptions: awf.submitOptsGenerator(wfUuid, priority),
	})
	if err != nil {
		return fmt.Errorf("failed to submit workflow: %s", err)
	}

	return nil
}

type ArgoClient struct {
	config        *config.ArgoConf
	ctx           context.Context
	apiClient     apiclient.Client
	serviceClient workflowpkg.WorkflowServiceClient
	namespace     string
	workflows     map[string]*ArgoWorkflow
}

func NewArgoClient(ctx context.Context, ac *config.ArgoConf, wfs []*config.Workflow) (*ArgoClient, error) {
	ctx, client, err := apiclient.NewClientFromOpts(
		apiclient.Opts{
			// right now only supports direct connection to k8s api
			// TODO: support more options here, see source for more info
			ClientConfigSupplier: func() clientcmd.ClientConfig { return ac.GetConfig() },
			Context:              ctx,
		})
	if err != nil {
		return nil, err
	}

	namespace, err := ac.GetNamespace()
	if err != nil {
		return nil, err
	}

	workflows := make(map[string]*ArgoWorkflow, len(wfs))
	for _, wf := range wfs {
		awf, err := ArgoWorkflowFromWorkflow(wf)
		if err != nil {
			return nil, err
		}

		workflows[wf.Id] = awf
	}

	// TODO: we should add a check for template method to the workflow struct,
	// then iterate through all our workflows here to validate that we have
	// templates defined in the cluster for each of them -- allows us to fail fast
	// if a template is missing.
	return &ArgoClient{
		config:        ac,
		ctx:           ctx,
		apiClient:     client,
		serviceClient: client.NewWorkflowServiceClient(),
		namespace:     namespace,
		workflows:     workflows,
	}, nil
}

func (ac *ArgoClient) SubmitWorkflow(
	ctx context.Context,
	wfId string,
	wfUuid uuid.UUID,
	priority int,
) error {
	wf, ok := ac.workflows[wfId]
	if !ok {
		return fmt.Errorf("unknown workflow '%s'", wfId)
	}

	return wf.SubmitWorkflow(ctx, ac, wfUuid, priority)
}

func (ac *ArgoClient) HandleAction(ctx context.Context, conn db.Conn, thread *db.Thread) error {
	handleFn := func() error {
		return ac.SubmitWorkflow(ctx, thread.WorkflowId, thread.Uuid, thread.Priority)
	}
	return HandleActionWrapper(ctx, conn, thread, true, handleFn)
}
