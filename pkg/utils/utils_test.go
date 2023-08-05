package utils

import (
	"reflect"
	"testing"
)

func TestConcat(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"d", "e", "f"}
	c := []string{"a", "b", "c", "d", "e", "f"}

	res := Concat(a, b)
	if !reflect.DeepEqual(res, c) {
		t.Fail()
	}
}
