package template

import (
	"io"
)

type Template struct {
	baseTemplate
}

func (t *Template) Execute(out io.Writer, data any) error {
	return t.template.Execute(out, data)
}

func (t *Template) ExecuteToString(data any) (string, error) {
	return t.executeToString(data)
}
