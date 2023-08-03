package caboose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/element84/swoop-go/pkg/s3"
	"github.com/gofrs/uuid/v5"
)

type S3 struct {
	s3 *s3.S3Driver
}

func NewS3(s3 *s3.S3Driver) *S3 {
	return &S3{s3}
}

func (s *S3) getJsonFromObject(ctx context.Context, key string) (any, error) {
	stream, err := s.s3.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	stat, err := stream.Stat()
	if err != nil {
		return nil, err
	}

	contentBytes := make([]byte, stat.Size)
	b, err := stream.Read(contentBytes)
	if int64(b) != stat.Size && err != nil {
		return nil, err
	}

	var j any
	err = json.Unmarshal(contentBytes, &j)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (s *S3) putJsonIntoObject(ctx context.Context, key string, j any) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(j)
	if err != nil {
		return err
	}

	opts := &s3.PutOptions{
		// allows us to preview in the minio console
		// application/json would be more appropriate but can't be previewed
		ContentType: "text/plain",
	}

	err = s.s3.Put(ctx, key, b, int64(b.Len()), opts)
	if err != nil {
		return err
	}

	return nil
}

func (s *S3) GetInput(ctx context.Context, workflowUuid uuid.UUID) (any, error) {
	key := fmt.Sprintf("/executions/%s/input.json", workflowUuid)
	return s.getJsonFromObject(ctx, key)
}

func (s *S3) GetOutput(ctx context.Context, workflowUuid uuid.UUID) (any, error) {
	key := fmt.Sprintf("/executions/%s/output.json", workflowUuid)
	return s.getJsonFromObject(ctx, key)
}

func (s *S3) PutWorkflowResource(ctx context.Context, workflowUuid uuid.UUID, json any) error {
	key := fmt.Sprintf("executions/%s/workflow.json", workflowUuid)
	return s.putJsonIntoObject(ctx, key, json)
}

func (s *S3) PutCallbackParams(ctx context.Context, callbackUuid uuid.UUID, json any) error {
	key := fmt.Sprintf("callbacks/%s/parameters.json", callbackUuid)
	return s.putJsonIntoObject(ctx, key, json)
}
