package config

import (
	"fmt"
)

type Conductors map[string]*Conductor

func (cs Conductors) setHandlers(handlers Handlers) error {
	for cName, c := range cs {
		c.Handlers = make([]*Handler, 0, len(c.HandlerNames))

		for _, hName := range c.HandlerNames {
			handler, ok := handlers[hName]
			if !ok {
				return fmt.Errorf(
					"cannot resolve handler '%s' for conductor '%s'",
					hName,
					cName,
				)
			}
			c.Handlers = append(c.Handlers, handler)
		}
	}

	return nil
}

type Conductor struct {
	HandlerNames []string   `yaml:"handlers,omitempty"`
	Handlers     []*Handler `yaml:"-"`
}

type SwoopConfig struct {
	Workflows  Workflows  `yaml:"workflows"`
	Handlers   Handlers   `yaml:"handlers"`
	Conductors Conductors `yaml:"conductors"`
}

func (sc *SwoopConfig) LinkAndValidate() error {
	// TODO: testing

	err := sc.Conductors.setHandlers(sc.Handlers)
	if err != nil {
		return err
	}

	err = sc.Workflows.setHandlers(sc.Handlers)
	if err != nil {
		return err
	}

	return nil
}

// TODO: using this pattern in many places
// -> make a generic unmarshal function that takes an interface with a postUnmarshal
//
//	function or something that can be used to run validation and other such steps
func (sc *SwoopConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type p SwoopConfig

	err := unmarshal((*p)(sc))
	if err != nil {
		return err
	}

	err = sc.LinkAndValidate()
	if err != nil {
		return err
	}

	return nil
}
