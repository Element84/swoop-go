package config

import (
	"fmt"

	"github.com/element84/swoop-go/pkg/config/jsonpath"
	"github.com/element84/swoop-go/pkg/utils"
)

type CallbackParameter struct {
	Value interface{}        `yaml:"value"`
	Path  *jsonpath.JsonPath `yaml:"path"`
}

type Callbacks map[string]*Callback

func (cbs *Callbacks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type p Callbacks

	err := unmarshal((*p)(cbs))
	if err != nil {
		return err
	}

	for cbName, cb := range *cbs {
		cb.Name = cbName
	}

	return nil
}

func (cbs *Callbacks) setHandlers(handlers Handlers) error {
	for _, cb := range *cbs {
		err := cb.setHandler(handlers)
		if err != nil {
			return err
		}
	}

	return nil
}

type Callback struct {
	Name        string       `yaml:"-"`
	HandlerName string       `yaml:"handler"`
	Type        CallbackType `yaml:"type"`
	// TODO: need to parse filter into a type
	FeatureFilter string              `yaml:"featureFilter,omitempty"`
	When          *CallbackWhen       `yaml:"when"`
	Parameters    *CallbackParameters `yaml:"parameters,omitempty"`
	Handler       *Handler            `yaml:"-"`
}

type CallbackParameters map[string]*CallbackParameter

func (cb *Callback) ValidateParams(params any) error {
	if cb.Handler == nil {
		return fmt.Errorf("No associated handler: '%s'", cb.HandlerName)
	}
	p := (*cb.Handler).Parameters
	return p.Validate(params)
}

func (cb *Callback) setHandler(handlers Handlers) error {
	err := cb._setHandler(handlers)
	if err != nil {
		return fmt.Errorf("callback '%s': %s", cb.Name, err)
	}

	return nil
}

func (cb *Callback) _setHandler(handlers Handlers) error {
	callbackParamNames := []string{}
	handlerParamNames := []string{}
	requiredParamNames := []string{}

	callbackParams := cb.Parameters
	handlerName := cb.HandlerName
	handler, ok := handlers[handlerName]
	if !ok {
		return fmt.Errorf(
			"cannot resolve handler '%s'",
			handlerName,
		)
	}

	cb.Handler = handler

	for name := range cb.Handler.Parameters.Properties {
		handlerParamNames = append(handlerParamNames, name)
		if utils.Contains(cb.Handler.Parameters.Required, name) {
			requiredParamNames = append(requiredParamNames, name)
		}
	}
	for name, param := range *callbackParams {
		callbackParamNames = append(callbackParamNames, name)

		if param.Path == nil {
			// if no jsonPath this is a value parameter we can validate
			schema, ok := cb.Handler.Parameters.Properties[name]
			if !ok {
				// later validation will catch that this param is not defined
				continue
			}

			err := schema.Validate(param.Value)
			if err != nil {
				return fmt.Errorf(
					"parameter '%s' bad value: %s",
					name,
					err,
				)
			}
		}
	}

	// Check if all required handler parameters are defined on callback
	for _, v := range requiredParamNames {
		if !utils.Contains(callbackParamNames, v) {
			return fmt.Errorf(
				"handler '%s' required parameter '%s' is not defined on callback",
				handlerName,
				v,
			)
		}
	}

	// Check all callback parameters are handler parameters
	for _, v := range callbackParamNames {
		if !utils.Contains(handlerParamNames, v) {
			return fmt.Errorf(
				"callback parameter '%s' is not a known parameter for handler '%s'",
				v,
				handlerName,
			)
		}
	}

	return nil
}
