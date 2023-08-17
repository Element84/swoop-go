package template

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func Test_UrlTemplate(t *testing.T) {
	data := map[string]any{
		"a": 213,
		"b": "val//b✅",
		"c": "$5ee😜+",
		"d": "path/compo+nent",
		"e": "unused",
	}
	yml := []byte(
		`url: "https://{{ .a }}:{{.b|urlquery}}@example.com/{{.d}}/endpoint?var={{ .c|urlquery }}&t=1234"`,
	)
	expected := "https://213:val%2F%2Fb%E2%9C%85@example.com/path/compo+nent/endpoint?var=%245ee%F0%9F%98%9C%2B&t=1234"

	s := struct {
		Url UrlTemplate `yaml:"url"`
	}{}

	err := yaml.Unmarshal(yml, &s)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}

	parsed, err := s.Url.Execute(data)
	if err != nil {
		t.Fatalf("error templating: %s", err)
	}

	if parsed.String() != expected {
		t.Fatalf("expected '%s', got '%s'", expected, parsed.String())
	}
}
