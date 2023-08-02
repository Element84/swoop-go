package config

import (
	"fmt"
	"strings"
)

type CallbackType string

const (
	SingleCallback     CallbackType = "single"
	PerFeatureCallback CallbackType = "perfeature"
)

var CallbackTypes = map[CallbackType]struct{}{
	SingleCallback:     {},
	PerFeatureCallback: {},
}

func (cbt CallbackType) String() string {
	return string(cbt)
}

func ParseCallbackType(s string) (CallbackType, error) {
	cbt := CallbackType(strings.ToLower(s))

	_, ok := CallbackTypes[cbt]
	if !ok {
		return "", fmt.Errorf("unknown callback type '%s'", s)
	}

	return cbt, nil
}

func (cbt *CallbackType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cbType string

	err := unmarshal(&cbType)
	if err != nil {
		return err
	}

	*cbt, err = ParseCallbackType(cbType)
	if err != nil {
		return err
	}

	return nil
}
