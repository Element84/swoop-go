package db_test

import (
	"testing"

	"github.com/gofrs/uuid/v5"

	"github.com/element84/swoop-go/pkg/context"
	"github.com/element84/swoop-go/pkg/states"

	dbtest "github.com/element84/swoop-go/pkg/utils/testing/db"

	. "github.com/element84/swoop-go/pkg/db"
)

func Test_Event(t *testing.T) {
	appName := "swoop-testing"
	ctx := context.NewApplicationContext(appName)

	wfUuid := uuid.Must(uuid.FromString("f44bb102-a200-4506-bdfb-6a238c33b22d"))

	testdb := dbtest.NewTestingDB(t, "db_event_")
	testdb.Create(ctx)
	conn, err := testdb.ConnectConfig().Connect(ctx)
	if err != nil {
		t.Fatalf("failed to get db connection: %s", err)
	}
	defer conn.Close(ctx)

	t.Run(
		"insert event",
		func(t *testing.T) {
			err := (&Event{
				ActionUuid: wfUuid,
				Status:     states.Queued,
			}).Insert(ctx, conn)
			if err != nil {
				t.Fatal(err)
			}

			t.Run(
				"check event source",
				func(t *testing.T) {
					var eventSrc string
					err := conn.QueryRow(
						ctx,
						"select event_source from swoop.event where action_uuid = $1",
						wfUuid,
					).Scan(&eventSrc)
					if err != nil {
						t.Fatal(err)
					}

					if eventSrc != appName {
						t.Fatalf("expected '%s', received '%s'", appName, eventSrc)
					}
				},
			)
		},
	)
}
