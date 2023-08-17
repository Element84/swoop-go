package template

import (
	"bytes"
	"text/template"
)

func executeTemplate(t *template.Template, data any) (string, error) {
	var out bytes.Buffer
	err := t.Execute(&out, data)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

type Template struct {
	raw      string
	template *template.Template
}

func (t *Template) makeTemplate() error {
	tmpl := template.New("")
	tmpl, err := tmpl.Parse(t.raw)
	if err != nil {
		return err
	}

	t.template = tmpl
	return nil
}

func (t *Template) Execute(data any) (string, error) {
	return executeTemplate(t.template, data)
}

func (t *Template) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	err := unmarshal(&s)
	if err != nil {
		return err
	}

	t.raw = s

	err = t.makeTemplate()
	if err != nil {
		return err
	}

	return nil
}
