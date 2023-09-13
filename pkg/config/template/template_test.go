package template

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func Test_Template_URL(t *testing.T) {
	t.Setenv("TEMPLATE_TEST_VAR_NAME", "8435848884422341")
	input := `{
		"person": "James",
		"question": {
			"noun": "speeding boat"
		},
		"other": "unused"
	}`
	yml := `template: "hello {{ .person }}, how is your {{ .question.noun }}? {{ env \"TEMPLATE_TEST_VAR_NAME\" }}"`
	expected := "hello James, how is your speeding boat? 8435848884422341"

	var data any
	err := json.Unmarshal([]byte(input), &data)
	if err != nil {
		t.Fatalf("failed to parse input json: %s", err)
	}

	s := struct {
		Template Template
	}{}

	err = yaml.Unmarshal([]byte(yml), &s)
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
