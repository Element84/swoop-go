package testing

import (
	"testing"
)

// not really sure how to validate the results in a portable way so these
// really just make sure we get a result, and leave it to the operator to
// review
func TestPathFromRoot(t *testing.T) {
	t.Log(PathFromRoot(t, ""))
}

func TestGetFixture(t *testing.T) {
	t.Log(GetFixture(t, "fixture"))
}
