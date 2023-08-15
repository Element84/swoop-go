package context

import (
	"context"
	"testing"
)

func Test_ApplicationContext(t *testing.T) {
	appName := "TestApp!"

	t.Run(
		"new app context",
		func(t *testing.T) {
			_ = NewApplicationContext(appName)
		},
	)

	t.Run(
		"retrieve application name",
		func(t *testing.T) {
			ctx := NewApplicationContext(appName)
			v, err := GetApplicationName(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if v != appName {
				t.Fatalf("expected '%s', recieved '%s'", appName, v)
			}
		},
	)

	t.Run(
		"bad context",
		func(t *testing.T) {
			ctx := context.Background()
			v, err := GetApplicationName(ctx)
			if err == nil {
				t.Fatalf("expected an error to be returned, but got value '%s'", v)
			}
		},
	)
}
