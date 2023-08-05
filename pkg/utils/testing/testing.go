package testing

import (
	"path"
	"runtime"
	"testing"
)

func PathFromRoot(t testing.TB, p string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve root directory")
	}

	return path.Join(path.Dir(filename), "..", "..", "..", p)
}

func GetFixture(t testing.TB, p string) string {
	return path.Join(PathFromRoot(t, "fixtures"), p)
}
