package config

import (
	"fmt"

	"github.com/element84/swoop-go/pkg/utils"
)

type Workflow struct {
	Callbacks map[string]*Callback `yaml:"callbacks"`
}

type Conductor struct {
	HandlerNames []string   `yaml:"handlers,omitempty"`
	Handlers     []*Handler `yaml:"-"`
}

type SwoopConfig struct {
	Workflows  map[string]*Workflow  `yaml:"workflows"`
	Handlers   map[string]*Handler   `yaml:"handlers"`
	Conductors map[string]*Conductor `yaml:"conductors"`
}

func (sc *SwoopConfig) LinkAndValidate() error {
	// TODO: break this func down into sub-functions
	// TODO: testing
	for handlerName, handler := range sc.Handlers {
		handler.Name = handlerName
	}

	for conductorName, conductor := range sc.Conductors {
		conductor.Handlers = make([]*Handler, 0, len(conductor.HandlerNames))

		for _, handlerName := range conductor.HandlerNames {
			handler, ok := sc.Handlers[handlerName]
			if !ok {
				return fmt.Errorf(
					"cannot resolve handler '%s' for conductor '%s'",
					handlerName,
					conductorName,
				)
			}
			conductor.Handlers = append(conductor.Handlers, handler)
		}
	}

	for wfName, wf := range sc.Workflows {
		for cbName, cb := range wf.Callbacks {
			var (
				callbackParamNames []string
				handlerParamNames  []string
				requiredParamNames []string
			)

			callbackParams := cb.Parameters
			handlerName := cb.HandlerName
			handler, ok := sc.Handlers[handlerName]
			if !ok {
				return fmt.Errorf(
					"cannot resolve handler '%s' for callback '%s' in workflow '%s'",
					handlerName,
					cbName,
					wfName,
				)
			}

			cb.Name = cbName
			cb.Handler = handler

			for name := range cb.Handler.Parameters.Properties {
				handlerParamNames = append(handlerParamNames, name)
				if utils.Contains(cb.Handler.Parameters.Required, name) {
					requiredParamNames = append(requiredParamNames, name)
				}
			}

			for name, param := range *callbackParams {
				callbackParamNames = append(callbackParamNames, name)

				if param.jsonPath == nil {
					// if no jsonPath this is a value parameter we can validate
					schema, ok := cb.Handler.Parameters.Properties[name]
					if !ok {
						// later validation will catch that this param is not defined
						continue
					}

					err := schema.Validate(param.Value)
					if err != nil {
						return fmt.Errorf(
							"workflow '%s' callback '%s' parameter '%s' bad value: %s",
							wfName,
							cbName,
							name,
							err,
						)
					}
				}
			}

			// Check if all required handler parameters are defined in callback
			for _, v := range requiredParamNames {
				if !utils.Contains(callbackParamNames, v) {
					return fmt.Errorf(
						"handler '%s' required parameter '%s' is not defined on callback '%s' in workflow '%s'",
						handlerName,
						v,
						cbName,
						wfName,
					)
				}
			}

			// Check all callback parameters are handler parameters
			for _, v := range callbackParamNames {
				if !utils.Contains(handlerParamNames, v) {
					return fmt.Errorf(
						"callback '%s' parameter '%s' is not a known parameter for handler '%s' in workflow '%s'",
						cbName,
						v,
						handlerName,
						wfName,
					)
				}
			}
		}
	}

	return nil
}

func (sc *SwoopConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type SC SwoopConfig
	var _sc SC

	err := unmarshal(&_sc)
	if err != nil {
		return err
	}

	nsc := SwoopConfig(_sc)

	err = nsc.LinkAndValidate()
	if err != nil {
		return err
	}

	*sc = nsc

	return nil
}
