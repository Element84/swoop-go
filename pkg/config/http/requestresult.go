package http

import (
	"fmt"
	"strings"

	"github.com/element84/swoop-go/pkg/errors"
)

type RequestResult string

const (
	Success RequestResult = "success"
	Error   RequestResult = "error"
	Fatal   RequestResult = "fatal"
)

var requestResults = map[RequestResult]struct{}{
	Success: {},
	Error:   {},
	Fatal:   {},
}

func (rt RequestResult) String() string {
	return string(rt)
}

func (rt RequestResult) ToError() error {
	var err error
	switch rt {
	case Success:
		err = nil
	case Error:
		err = errors.NewRequestError(fmt.Errorf("request response was error"), true)
	case Fatal:
		err = errors.NewRequestError(fmt.Errorf("request response was fatal error"), false)
	}
	return err
}

func RequestResultFromError(err error) (RequestResult, bool) {
	if err == nil {
		return Success, true
	}

	re, ok := err.(*errors.RequestError)
	if !ok {
		return "", false
	}

	if re.Retryable {
		return Error, true
	}

	return Fatal, true
}

func parseRequestResult(s string) (RequestResult, error) {
	rt := RequestResult(strings.ToLower(s))

	_, ok := requestResults[rt]
	if !ok {
		return "", fmt.Errorf("unknown request result '%s'", s)
	}

	return rt, nil
}

func (rt *RequestResult) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rType string

	err := unmarshal(&rType)
	if err != nil {
		return err
	}

	*rt, err = parseRequestResult(rType)
	if err != nil {
		return err
	}

	return nil
}
