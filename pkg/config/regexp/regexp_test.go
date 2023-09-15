package regexp_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	. "github.com/element84/swoop-go/pkg/config/regexp"
)

func Test_Regexp(t *testing.T) {
	match := "this 9rth string ?? matches"
	nomatch := " string does not match "
	conf := "is .+ string \\?{2} match"

	rx := &Regexp{}

	err := yaml.Unmarshal([]byte(conf), rx)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}

	t.Run(
		"match",
		func(t *testing.T) {
			if !rx.MatchString(match) {
				t.Fatal("should have matched string, but didn't")
			}
		},
	)

	t.Run(
		"nomatch",
		func(t *testing.T) {
			if rx.MatchString(nomatch) {
				t.Fatal("should not have matched string, but did")
			}
		},
	)
}
