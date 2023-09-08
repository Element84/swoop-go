package http

import (
	"fmt"
	"strings"
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
