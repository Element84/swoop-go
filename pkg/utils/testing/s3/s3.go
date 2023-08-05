package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/s3"
	test "github.com/element84/swoop-go/pkg/utils/testing"
)

type TestingS3 struct {
	test       testing.TB
	Driver     *s3.S3Driver
	JsonClient *s3.JsonClient
}

func NewTestingS3(test testing.TB, prefix string) *TestingS3 {
	Driver := &s3.S3Driver{
		Bucket:   strings.ToLower(prefix + test.Name()),
		Endpoint: os.Getenv("SWOOP_S3_ENDPOINT"),
	}
	return &TestingS3{test, Driver, s3.NewJsonClient(Driver)}
}

func (t *TestingS3) putFixture(ctx context.Context, key string, path string) error {
	jsonFile, err := os.Open(path)
	if err != nil {
		return err
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	j := map[string]interface{}{}
	err = json.Unmarshal(byteValue, &j)
	if err != nil {
		return err
	}

	return t.JsonClient.PutJsonIntoObject(ctx, key, j)
}

func (t *TestingS3) PutInput(ctx context.Context, workflowUuid uuid.UUID) {
	key := fmt.Sprintf("executions/%s/input.json", workflowUuid)
	err := t.putFixture(ctx, key, test.GetFixture(t.test, "payloads/input.json"))
	if err != nil {
		t.test.Fatalf("failed to put input to key '%s': %s", key, err)
	}
}

func (t *TestingS3) PutOutput(ctx context.Context, workflowUuid uuid.UUID) {
	key := fmt.Sprintf("executions/%s/output.json", workflowUuid)
	err := t.putFixture(ctx, key, test.GetFixture(t.test, "payloads/output.json"))
	if err != nil {
		t.test.Fatalf("failed to put output to key '%s': %s", key, err)
	}
}

func (t *TestingS3) SetupBucket(ctx context.Context) {
	cleanup := func() {
		_ = t.Driver.RemoveBucket(ctx)
	}
	t.test.Cleanup(cleanup)

	err := t.Driver.MakeBucket(ctx)
	if err != nil {
		t.test.Fatalf("failed to make bucket: %s", err)
	}

}
