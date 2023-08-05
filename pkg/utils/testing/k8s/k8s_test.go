package k8s

import (
	"context"
	"testing"
)

func TestMakeNamespace(t *testing.T) {
	_ = TestNamespaceAndConfigFlags(context.Background(), t, "testing-k8s-")
}
