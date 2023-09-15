package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/element84/swoop-go/pkg/config/jsonpath"
	"github.com/element84/swoop-go/pkg/config/regexp"
)

type Response struct {
	StatusCode int
	Body       string
	Json       any `json:"-"`
}

func wrapRequest(resp *http.Response, err error) (*Response, error) {
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// TODO: check content type before trying this
	bodyJson := map[string]any{}
	err = json.Unmarshal([]byte(body), &bodyJson)
	if err != nil {
		// not a json body, no worries, just set to nil
		bodyJson = nil
	}

	return &Response{
		resp.StatusCode,
		string(body),
		bodyJson,
	}, nil
}

type responseMatcher struct {
	StatusCode int                `yaml:"statusCode"`
	JsonPath   *jsonpath.JsonPath `yaml:"jsonPath"`
	Message    *regexp.Regexp     `yaml:"message"`
	Result     RequestResult      `yaml:"result"`
}

func (rm *responseMatcher) match(resp *Response) (matched bool, err error) {
	if rm.StatusCode != resp.StatusCode {
		return false, nil
	}
	if rm.Message != nil && !rm.Message.MatchString(resp.Body) {
		return false, nil
	}
	if rm.JsonPath != nil {
		if !rm.JsonPath.Has(resp.Json) {
			return false, nil
		}
	}

	return true, rm.Result.ToError()
}

type responseChecker []*responseMatcher

func (rc *responseChecker) check(resp *Response) error {
	for _, matcher := range *rc {
		matched, err := matcher.match(resp)
		if matched {
			return err
		}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return Error.ToError()
}
