package template

import (
	"net/url"
)

type UrlTemplate struct {
	baseTemplate
}

func (t *UrlTemplate) Execute(data any) (*url.URL, error) {
	s, err := t.executeToString(data)
	if err != nil {
		return nil, err
	}

	return url.Parse(s)
}
