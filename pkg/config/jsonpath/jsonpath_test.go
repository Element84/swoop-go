package jsonpath_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	. "github.com/element84/swoop-go/pkg/config/jsonpath"
)

func Test_JsonPathUnmarshalBad(t *testing.T) {
	conf := "not a valid jsonpath"

	jp := &JsonPath{}

	err := yaml.Unmarshal([]byte(conf), jp)
	if err == nil {
		t.Fatal("should have failed to parse bad jsonpath string")
	}
}

func Test_JsonPathUnmarshalGood(t *testing.T) {
	conf := "b[?(@.y > 10)].x"

	jp := &JsonPath{}

	err := yaml.Unmarshal([]byte(conf), jp)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}

	if jp.String() != conf {
		t.Fatalf("expected parsed to be '%s', but was '%s'", conf, jp)
	}
}
