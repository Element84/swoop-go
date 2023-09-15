package http

import (
	"bytes"
	"context"
	"net/http"

	"github.com/creasty/defaults"

	"github.com/element84/swoop-go/pkg/config/template"
)

type Client struct {
	Url             *template.UrlTemplate         `yaml:"url"`
	Method          HttpMethod                    `yaml:"method"`
	Body            *template.Template            `yaml:"body"`
	Headers         map[string]*template.Template `yaml:"headers"`
	ResponseChecker *responseChecker              `yaml:"responses"`
	Follow          bool                          `default:"true" yaml:"followRedirects"`
	Transport       *http.Transport               `yaml:"transport"`
	client          *http.Client
}

func (s *Client) NewRequest(data any) (*http.Request, error) {
	url, err := s.Url.Execute(data)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	err = s.Body.Execute(&body, data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(s.Method.String(), url.String(), &body)
	if err != nil {
		return nil, err
	}

	for headerName, headerTemplate := range s.Headers {
		headerValue, err := headerTemplate.ExecuteToString(data)
		if err != nil {
			return nil, err
		}

		req.Header.Set(headerName, headerValue)
	}

	return req, nil
}

func (s *Client) MakeRequest(ctx context.Context, req *http.Request) error {
	resp, err := wrapRequest(s.client.Do(req.WithContext(ctx)))
	if err != nil {
		// TODO: how to differentiate between terminal and transient errors in client
		return err
	}
	return s.ResponseChecker.check(resp)
}

func (s *Client) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(s)

	type p Client

	err := unmarshal((*p)(s))
	if err != nil {
		return err
	}

	s.client = &http.Client{}

	if s.Transport != nil {
		s.client.Transport = s.Transport
	}

	if !s.Follow {
		s.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// TODO: ensure we have required fields
	// separate validate method?

	return nil
}
