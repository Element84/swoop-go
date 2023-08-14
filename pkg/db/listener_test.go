package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/element84/swoop-go/pkg/db"
)

type TestHandler struct {
	name    string
	channel chan string
}

func (t *TestHandler) GetName() string {
	return t.name
}

func (t *TestHandler) Notify() {
	t.channel <- t.name
}

func TestListener(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()

	// we don't currently need an explicit test database as
	// notifications do not depend on or modify db state
	dbconf := &ConnectConfig{}

	notifierConn, err := dbconf.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to create database connection for notifier: %s", err)
	}

	notifications := make(chan string, 1)

	receiveNotification := func() string {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer func() {
			cancel()
		}()

		select {
		case <-ctx.Done():
			return "[timed out]"
		case msg := <-notifications:
			return msg
		}
	}

	handlers := []*TestHandler{
		{name: "h1", channel: notifications},
		{name: "h2", channel: notifications},
		{name: "h3", channel: notifications},
	}

	err = Listen(ctx, dbconf, handlers)
	if err != nil {
		t.Fatalf("listening failed: %s", err)
	}

	testCases := [][]string{
		{"h1", "h1"},
		{"h2", "h2"},
		{"h1", "h1"},
		{"h4", "[timed out]"},
		{"h3", "h3"},
		{"h1", "h1"},
	}

	for _, val := range testCases {
		send := val[0]
		expected := val[1]
		t.Run(
			fmt.Sprintf("notify %s, expect %s", send, expected),
			func(t *testing.T) {
				sendReceive := func(channel string) string {
					_, err := notifierConn.Exec(ctx, "select pg_notify($1, $1)", channel)
					if err != nil {
						t.Fatalf("failed to notify: %s", err)
					}

					return receiveNotification()
				}

				received := sendReceive(send)
				if received != expected {
					t.Fatalf("recieved value not expected value: '%s' != '%s'", received, expected)
				}

			},
		)
	}
}

func TestListenerNoHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expected := "not listening: nothing to listen to"
	err := Listen(ctx, &ConnectConfig{}, []Notifiable{})
	if err == nil {
		t.Fatal("should have thrown error for no notifiables")
	} else if err.Error() != expected {
		t.Fatalf("unexpected error: '%s'; wanted: '%s'", err, expected)
	}
}
