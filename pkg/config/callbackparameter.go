package config

import (
	"github.com/ohler55/ojg/jp"
)

type CallbackParameter struct {
	Value    interface{} `yaml:"value"`
	Path     string      `yaml:"path"`
	jsonPath *jp.Expr
}

func (p *CallbackParameter) HasPath() bool {
	return p.jsonPath != nil
}

func (p *CallbackParameter) GetJsonPath(data any) any {
	return p.jsonPath.Get(data)
}

func (p *CallbackParameter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// inherit the type so we can use it for unmarshaling
	// but so it has a different name and we can cast it later
	// if we don't do this we get an error "excessive aliasing"
	type CP CallbackParameter
	var cp CP

	err := unmarshal(&cp)
	if err != nil {
		return err
	}

	// cast it to the real type
	np := CallbackParameter(cp)

	// do the extra init stuff we need for a path
	if np.Path != "" {
		jsonPath, err := jp.ParseString(np.Path)
		if err != nil {
			return err
		}
		np.jsonPath = &jsonPath
	}

	// replace the allocated instance with ours
	*p = np

	return nil
}
