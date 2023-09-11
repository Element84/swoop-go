package config

import (
	"fmt"
	"strings"
)

type HandlerType string

const (
	Noop          HandlerType = "noop"
	SyncHttp      HandlerType = "synchttp"
	ArgoWorkflows HandlerType = "argoworkflows"
	Cirrus        HandlerType = "cirrus"
)

var HandlerTypes = map[HandlerType]struct{}{
	Noop:          {},
	SyncHttp:      {},
	ArgoWorkflows: {},
	Cirrus:        {},
}

func (cbt HandlerType) String() string {
	return string(cbt)
}

func ParseHandlerType(s string) (HandlerType, error) {
	ht := HandlerType(strings.ToLower(s))

	_, ok := HandlerTypes[ht]
	if !ok {
		return "", fmt.Errorf("unknown handler type '%s'", s)
	}

	return ht, nil
}

func (ht *HandlerType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var hType string

	err := unmarshal(&hType)
	if err != nil {
		return err
	}

	*ht, err = ParseHandlerType(hType)
	if err != nil {
		return err
	}

	return nil
}
