package http

import (
	"fmt"
	"net/http"
	"strings"
)

type HttpMethod string

const (
	GET    HttpMethod = http.MethodGet
	POST              = http.MethodPost
	PUT               = http.MethodPut
	PATCH             = http.MethodPatch
	DELETE            = http.MethodDelete
)

var supportedHttpMethods = map[HttpMethod]struct{}{
	GET:    {},
	POST:   {},
	PUT:    {},
	PATCH:  {},
	DELETE: {},
}

func (hm HttpMethod) String() string {
	return string(hm)
}

func parseHttpMethod(s string) (HttpMethod, error) {
	hm := HttpMethod(strings.ToUpper(s))

	_, ok := supportedHttpMethods[hm]
	if !ok {
		return "", fmt.Errorf("unknown or unsupported HTTP method type '%s'", s)
	}

	return hm, nil
}

func (hm *HttpMethod) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var h string

	err := unmarshal(&h)
	if err != nil {
		return err
	}

	*hm, err = parseHttpMethod(h)
	if err != nil {
		return err
	}

	return nil
}
