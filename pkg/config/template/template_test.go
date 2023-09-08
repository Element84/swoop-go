package template

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func Test_Template_URL(t *testing.T) {
	data := map[string]any{
		"person": "James",
		"question": map[string]any{
			"noun": "speeding boat",
		},
		"other": "unused",
	}
	yml := []byte(
		`template: "hello {{ .person }}, how is your {{ .question.noun }}?"`,
	)
	expected := "hello James, how is your speeding boat?"

	s := struct {
		Template Template
	}{}

	err := yaml.Unmarshal(yml, &s)
	if err != nil {
		t.Fatalf("error parsing yaml: %s", err)
	}

	parsed, err := s.Template.ExecuteToString(data)
	if err != nil {
		t.Fatalf("error templating: %s", err)
	}

	if parsed != expected {
		t.Fatalf("expected '%s', got '%s'", expected, parsed)
	}
}
