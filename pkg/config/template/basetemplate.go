package template

import (
	"bytes"
	"os"
	"text/template"
)

var (
	fnMap = map[string]interface{}{
		// can call this function like {{ env "VAR_NAME" }} in a template
		"env": os.Getenv,
	}
)

type baseTemplate struct {
	template *template.Template
}

func newTemplate(t string) (*baseTemplate, error) {
	tmpl := template.New("").Funcs(template.FuncMap(fnMap))
	tmpl, err := tmpl.Parse(t)
	if err != nil {
		return nil, err
	}

	return &baseTemplate{template: tmpl}, nil
}

func (t *baseTemplate) executeToString(data any) (string, error) {
	var out bytes.Buffer
	err := t.template.Execute(&out, data)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func (t *baseTemplate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	err := unmarshal(&s)
	if err != nil {
		return err
	}

	tmpl, err := newTemplate(s)
	if err != nil {
		return err
	}
	*t = *tmpl

	return nil
}
