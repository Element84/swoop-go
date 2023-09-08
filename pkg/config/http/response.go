package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/element84/swoop-go/pkg/config/jsonpath"
	"github.com/element84/swoop-go/pkg/config/regexp"
)

type response struct {
	StatusCode int
	Body       string
	Json       any
}

func wrapRequest(resp *http.Response, err error) (*response, error) {
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

	return &response{
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

func (rm *responseMatcher) match(resp *response) (result RequestResult, matched bool) {
	if rm.StatusCode != resp.StatusCode {
		return "", false
	}
	if rm.Message != nil && !rm.Message.MatchString(resp.Body) {
		return "", false
	}
	if rm.JsonPath != nil {
		if !rm.JsonPath.Has(resp.Json) {
			return "", false
		}
	}

	return rm.Result, true
}

type responseChecker []*responseMatcher

func (rc *responseChecker) check(resp *response) RequestResult {
	for _, matcher := range *rc {
		result, matched := matcher.match(resp)
		if matched {
			return result
		}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return Success
	}

	return Error
}
