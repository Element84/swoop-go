package db_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"

	dbtest "github.com/element84/swoop-go/pkg/utils/testing/db"

	. "github.com/element84/swoop-go/pkg/db"
)

func Test_GetProcessableActions(t *testing.T) {
	ctx := context.Background()
	wfUuid := uuid.Must(uuid.FromString("f44bb102-a200-4506-bdfb-6a238c33b22d"))

	testdb := dbtest.NewTestingDB(t, "db_thread_")
	testdb.Create(ctx)
	conn, err := testdb.ConnectConfig().Connect(ctx)
	if err != nil {
		t.Fatalf("failed to get db connection: %s", err)
	}
	defer conn.Close(ctx)

	conn2, err := testdb.ConnectConfig().Connect(ctx)
	if err != nil {
		t.Fatalf("failed to get db connection: %s", err)
	}
	defer conn2.Close(ctx)

	// stage actions
	for i := 0; i < 300; i++ {
		_, err := NewCallbackAction("callback", "handler", "type", wfUuid).Insert(ctx, conn)
		if err != nil {
			t.Fatalf("failed inserting callback action iteration #%d: %s", i, err)
		}
	}

	uuids := []uuid.UUID{}
	threads := []*Thread{}
	for i := 0; i < 12; i++ {
		ths, err := GetProcessableThreads(ctx, conn, "handler", 29, uuids)
		if err != nil {
			t.Fatalf("failed getting threads iteration #%d: %s", i, err)
		}

		for _, thread := range ths {
			uuids = append(uuids, thread.Uuid)
			threads = append(threads, thread)
		}
	}

	t.Run(
		"check len(fetched) == expected",
		func(t *testing.T) {
			expected := 300
			if len(threads) != expected {
				t.Fatalf("Fetched thread count not expected count: %d != %d", len(threads), expected)
			}
		},
	)

	t.Run(
		"check no more threads available",
		func(t *testing.T) {
			ths, err := GetProcessableThreads(ctx, conn2, "handler", 29, []uuid.UUID{})
			if err != nil {
				t.Fatalf("failed getting threads: %s", err)
			}

			if len(ths) > 0 {
				t.Fatal("got a thread when none should be visible")
			}
		},
	)

	t.Run(
		"check releasing lock",
		func(t *testing.T) {
			thread := threads[0]
			err := thread.Unlock(ctx, conn)
			if err != nil {
				t.Fatalf("failed unlocking thread: %s", err)
			}

			ths, err := GetProcessableThreads(ctx, conn2, "handler", 29, []uuid.UUID{})
			if err != nil {
				t.Fatalf("failed getting threads: %s", err)
			}

			expected := 1
			if len(ths) != expected {
				t.Fatalf("Fetched thread count not expected count: %d != %d", len(ths), expected)
			}

			if ths[0].Uuid != thread.Uuid {
				t.Fatalf("Fetched unexpected thread: '%s' != '%s'", threads[0].Uuid, thread.Uuid)
			}
		},
	)

	t.Run(
		"check event inserts",
		func(t *testing.T) {
			thread := threads[1]

			t.Run(
				"queued",
				func(t *testing.T) {
					err := thread.InsertQueuedEvent(ctx, conn)
					if err != nil {
						t.Fatalf("failed to insert queued event: %s", err)
					}
				},
			)
			t.Run(
				"successful",
				func(t *testing.T) {
					err := thread.InsertSuccessfulEvent(ctx, conn)
					if err != nil {
						t.Fatalf("failed to insert successful event: %s", err)
					}
				},
			)
			t.Run(
				"failed",
				func(t *testing.T) {
					err := thread.InsertFailedEvent(ctx, conn, "error")
					if err != nil {
						t.Fatalf("failed to insert failed event: %s", err)
					}
				},
			)
			t.Run(
				"retries exhausted",
				func(t *testing.T) {
					err := thread.InsertRetriesExhaustedEvent(ctx, conn, "no more retries")
					if err != nil {
						t.Fatalf("failed to insert retries exhausted event: %s", err)
					}
				},
			)
			t.Run(
				"backoff",
				func(t *testing.T) {
					err := thread.InsertBackoffEvent(ctx, conn, 605, "error")
					if err != nil {
						t.Fatalf("failed to insert backoff event: %s", err)
					}
				},
			)
		},
	)
}
