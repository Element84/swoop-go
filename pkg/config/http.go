package config

import (
	"net/http"

	"github.com/creasty/defaults"

	"github.com/element84/swoop-go/pkg/config/template"
)

type ResponseMatch struct {
	Status int `yaml:"status"`
	// TODO: regex?
	// TODO: how to match values a json payload (jsonpath condition?)
	//       -> a string can be valid json,
	//          could match content type and wrap in "" before applying jsonpath condition
	Message string `yaml:"message"`
	// TODO: needs to be something like success, failure, fatal? Call it type?
	Fatal bool `yaml:"fatal"`
}

// TODO: probably needs to be moved to own file
// needs to be reconsidered entirely, i.e., how do we match on responses
// needs a method to match a response, with a default behavior of 2xx good, anything else bad
type ResponseChecker []*ResponseMatch

// TODO: return type is same as ResponseMatch type (proposed above)
func (rc *ResponseChecker) Check(response *http.Response) bool {
	// TODO: implement this
	return false
}

// IDEA: should we support fetching values from the env for use templating?
type HttpRequest struct {
	Url    *template.UrlTemplate `yaml:"url"`
	Method HttpMethod            `yaml:"method"`
	Body   *template.Template    `yaml:"body"`
	// TODO: should content type be dynamically set if not provided?
	// see https://pkg.go.dev/net/http#DetectContentType
	ContentType     string                        `default:"text/plain; charset=UTF-8" yaml:"contentType"`
	Headers         map[string]*template.Template `yaml:"headers"`
	ResponseChecker *ResponseChecker              `yaml:"responses"`
	Follow          bool                          `default:"true" yaml:"followRedirects"`
	Transport       *http.Transport               `yaml:"transport"`
	client          *http.Client
}

// TODO: method to make request

// TODO: test HttpRequest works against test server
//       see https://www.digitalocean.com/community/tutorials/how-to-make-http-requests-in-go

func (s *HttpRequest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(s)

	type p HttpRequest

	err := unmarshal((*p)(s))
	if err != nil {
		return err
	}

	s.client = &http.Client{Transport: s.Transport}

	if !s.Follow {
		s.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// TODO: ensure we have required fields
	// separate validate method?

	return nil
}

/*
1 - validation
2 - templating
3 - test making request against test server
4 - response validation
*/
