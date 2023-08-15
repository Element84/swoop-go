package config

import (
	"bytes"
	"encoding/json"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Handler struct {
	Name        string             `yaml:"-"`
	Type        string             `yaml:"type,omitempty"`
	Url         string             `yaml:"url,omitempty"`
	RequestBody string             `yaml:"requestBody,omitempty"`
	Operation   string             `yaml:"operation,omitempty"`
	Secrets     []*Secret          `yaml:"secrets,omitempty"`
	Headers     map[string]string  `yaml:"headers,omitempty"`
	Backoff     map[string]int     `yaml:"backoff,omitempty"`
	Errors      []*Error           `yaml:"errors,omitempty"`
	Parameters  *HandlerParameters `yaml:"parameters"`
}

type Secret struct {
	Name string `yaml:"name,omitempty"`
	Type string `yaml:"type,omitempty"`
	Path string `yaml:"path,omitempty"`
	TTL  int    `yaml:"ttl,omitempty"`
}

type Error struct {
	Status    int    `yaml:"status,omitempty"`
	Message   string `yaml:"message,omitempty"`
	Retryable string `yaml:"retryable,omitempty"`
}

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
