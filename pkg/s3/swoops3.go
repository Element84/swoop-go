package s3

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
)

type SwoopS3 struct {
	jsonClient *JsonClient
}

func NewSwoopS3(jsonClient *JsonClient) *SwoopS3 {
	return &SwoopS3{jsonClient}
}

func (s *SwoopS3) GetInput(ctx context.Context, workflowUuid uuid.UUID) (any, error) {
	key := fmt.Sprintf("/executions/%s/input.json", workflowUuid)
	return s.jsonClient.GetJsonFromObject(ctx, key)
}

func (s *SwoopS3) GetOutput(ctx context.Context, workflowUuid uuid.UUID) (any, error) {
	key := fmt.Sprintf("/executions/%s/output.json", workflowUuid)
	return s.jsonClient.GetJsonFromObject(ctx, key)
}

func (s *SwoopS3) PutWorkflowResource(ctx context.Context, workflowUuid uuid.UUID, json any) error {
	key := fmt.Sprintf("executions/%s/workflow.json", workflowUuid)
	return s.jsonClient.PutJsonIntoObject(ctx, key, json)
}

func (s *SwoopS3) PutCallbackParams(ctx context.Context, callbackUuid uuid.UUID, json any) error {
	key := fmt.Sprintf("callbacks/%s/parameters.json", callbackUuid)
	return s.jsonClient.PutJsonIntoObject(ctx, key, json)
}

func (s *SwoopS3) GetCallbackParams(ctx context.Context, callbackUuid uuid.UUID) (any, error) {
	key := fmt.Sprintf("callbacks/%s/parameters.json", callbackUuid)
	return s.jsonClient.GetJsonFromObject(ctx, key)
}

func (s *SwoopS3) PutCallbackHttp(ctx context.Context, callbackUuid uuid.UUID, json any) error {
	key := fmt.Sprintf("callbacks/%s/http.json", callbackUuid)
	return s.jsonClient.PutJsonIntoObject(ctx, key, json)
}
