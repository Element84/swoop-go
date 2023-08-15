package context

import (
	"context"
	"fmt"
)

type swoopApplicationKey string

const (
	key = swoopApplicationKey("app")
)

func NewApplicationContext(applicationName string) context.Context {
	return context.WithValue(context.Background(), key, applicationName)
}

func GetApplicationName(ctx context.Context) (string, error) {
	val := ctx.Value(key)
	if val == nil {
		return "", fmt.Errorf("application name not found in context")
	}
	return val.(string), nil
}
