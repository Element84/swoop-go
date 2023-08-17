package template

import (
	"net/url"
)

type UrlTemplate struct {
	Template
}

func (t *UrlTemplate) Execute(data any) (*url.URL, error) {
	s, err := executeTemplate(t.template, data)
	if err != nil {
		return nil, err
	}

	return url.Parse(s)
}
