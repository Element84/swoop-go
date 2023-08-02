package caboose

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
	"github.com/element84/swoop-go/pkg/states"
)

func putFixture(ctx context.Context, s3 *S3, key string, path string) error {
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

	return s3.putJsonIntoObject(ctx, key, j)
}

func putInput(ctx context.Context, s3 *S3, workflowUuid uuid.UUID, path string) error {
	key := fmt.Sprintf("executions/%s/input.json", workflowUuid)
	return putFixture(ctx, s3, key, path)
}

func putOutput(ctx context.Context, s3 *S3, workflowUuid uuid.UUID, path string) error {
	key := fmt.Sprintf("executions/%s/output.json", workflowUuid)
	return putFixture(ctx, s3, key, path)
}

func TestCallbacks(t *testing.T) {
	ctx := context.Background()
	bucket := "callbacktestbucket"
	wfName := "mirror"
	status, _ := states.Parse("successful")
	wfProps := &WorkflowProperties{
		Uuid:   uuid.Must(uuid.FromString("f44bb102-a200-4506-bdfb-6a238c33b22d")),
		Status: states.Successful,
	}

	// TODO better way to reference fixture paths
	conf, err := config.Parse("../../fixtures/swoop-config.yml")
	if err != nil {
		t.Fatalf("failed to parse config file: %s", err)
	}

	callbacks, _ := MapConfigCallbacks(conf).lookup(wfName, states.FinalState(status))

	driver := &s3.S3Driver{
		Bucket:   bucket,
		Endpoint: os.Getenv("SWOOP_S3_ENDPOINT"),
	}

	defer func() {
		err := driver.RemoveBucket(ctx)
		if err != nil {
			t.Fatalf("failed to remove bucket: %s", err)
		}
	}()

	err = driver.MakeBucket(ctx)
	if err != nil {
		t.Fatalf("failed to make bucket: %s", err)
	}

	s3 := NewS3(driver)

	err = putInput(ctx, s3, wfProps.Uuid, "../../fixtures/payloads/input.json")
	if err != nil {
		t.Fatalf("failed to put input: %s", err)
	}

	err = putOutput(ctx, s3, wfProps.Uuid, "../../fixtures/payloads/output.json")
	if err != nil {
		t.Fatalf("failed to put output: %s", err)
	}

	db, err := db.Connect(ctx)

	cbx := NewCallbackExecutor(ctx, s3, db)

	err = cbx.ProcessCallbacks(callbacks, wfProps)
	if err != nil {
		t.Fatalf("failed to process callbacks: %s", err)
	}
}
