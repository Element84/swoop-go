package config

import (
	"bytes"
	"encoding/json"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type HandlerParameters jsonschema.Schema

func (p *HandlerParameters) Validate(data any) error {
	j := jsonschema.Schema(*p)
	return j.Validate(data)
}

func (p *HandlerParameters) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		params map[string]map[string]interface{}
	)

	reqrd := []string{}

	err := unmarshal(&params)
	if err != nil {
		return err
	}

	for name, config := range params {
		_, ok := config["default"]
		if !ok {
			reqrd = append(reqrd, name)
		}
	}

	j := map[string]interface{}{
		"type":       "object",
		"properties": params,
		"required":   reqrd,
	}

	// remarshal schema to string for compiling
	// unfortunately it can't compile from the unmarshaled yaml
	schema, err := json.Marshal(j)
	if err != nil {
		return err
	}

	compiler := jsonschema.NewCompiler()
	compiler.ExtractAnnotations = true
	err = compiler.AddResource("/parameters", bytes.NewReader(schema))
	if err != nil {
		return err
	}

	Schema, err := compiler.Compile("/parameters")
	if err != nil {
		return err
	}

	*p = HandlerParameters(*Schema)

	return nil
}
