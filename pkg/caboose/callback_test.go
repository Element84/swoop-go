package caboose

import (
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/s3"
	"github.com/element84/swoop-go/pkg/states"

	"github.com/element84/swoop-go/pkg/utils/testing/config"
	"github.com/element84/swoop-go/pkg/utils/testing/db"
	testS3 "github.com/element84/swoop-go/pkg/utils/testing/s3"
)

func TestCallbacks(t *testing.T) {
	ctx := context.Background()
	wfName := "mirror"
	status, _ := states.Parse("successful")
	wfProps := &WorkflowProperties{
		Uuid:   uuid.Must(uuid.FromString("f44bb102-a200-4506-bdfb-6a238c33b22d")),
		Status: states.Successful,
	}

	conf := config.LoadConfigFixture(t)
	callbacks, _ := MapConfigCallbacks(conf).Lookup(wfName, states.FinalState(status))

	t3 := testS3.NewTestingS3(t, "caboose-callbacks-")
	t3.SetupBucket(ctx)
	t3.PutInput(ctx, wfProps.Uuid)
	t3.PutOutput(ctx, wfProps.Uuid)

	testdb := db.NewTestingDB(t, "caboose_callbacks_")
	testdb.Create(ctx)
	db, err := testdb.Conf.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to get db connection: %s", err)
	}

	cbx := NewCallbackExecutor(ctx, s3.NewSwoopS3(t3.JsonClient), db)

	err = cbx.ProcessCallbacks(callbacks, wfProps)
	if err != nil {
		t.Fatalf("failed to process callbacks: %s", err)
	}
}
