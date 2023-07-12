package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type SwoopConfig struct {
	Workflows map[string]Workflow `yaml:"workflows"`
	Handlers  map[string]Handler  `yaml:"handlers"`
}

// Callback structs
type CallbackParameter struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Callback struct {
	Handler       string              `yaml:"handler"`
	Type          string              `yaml:"type"`
	FeatureFilter string              `yaml:"feature_filter,omitempty"`
	When          string              `yaml:"when,omitempty"`
	Enabled       bool                `yaml:"enabled,omitempty"`
	Parameters    []CallbackParameter `yaml:"parameters,omitempty"`
}

type Workflow struct {
	Callbacks map[string]Callback `yaml:"callbacks"`
}

// Handler structs
type Handler struct {
	Type        string             `yaml:"type,omitempty"`
	Url         string             `yaml:"url,omitempty"`
	RequestBody string             `yaml:"requestBody,omitempty"`
	Operation   string             `yaml:"operation,omitempty"`
	Secrets     []Secret           `yaml:"secrets,omitempty"`
	Headers     []Header           `yaml:"headers,omitempty"`
	Backoff     map[string]int     `yaml:"backoff,omitempty"`
	Errors      []Error            `yaml:"errors,omitempty"`
	Parameters  []HandlerParameter `yaml:"parameters,omitempty"`
}

type Secret struct {
	Name string `yaml:"name,omitempty"`
	Type string `yaml:"type,omitempty"`
	Path string `yaml:"path,omitempty"`
	TTL  int    `yaml:"ttl,omitempty"`
}

type Header struct {
	Name          string `yaml:"name,omitempty"`
	Value         string `yaml:"value,omitempty"`
	ContentType   string `yaml:"Content-Type,omitempty"`
	XWorkflowName string `yaml:"X-Workflow-Name,omitempty"`
}

type Error struct {
	Status    int    `yaml:"status,omitempty"`
	Message   string `yaml:"message,omitempty"`
	Retryable string `yaml:"retryable,omitempty"`
}

type HandlerParameter struct {
	Name    string `yaml:"name,omitempty"`
	Default string `yaml:"default,omitempty"`
}

func loadYaml(inputFile string, conf *SwoopConfig) error {
	readFile, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(readFile, conf)
	if err != nil {
		return err
	}

	return nil
}

func Parse(configFile string) (*SwoopConfig, error) {
	conf := &SwoopConfig{}
	err := loadYaml(configFile, conf)
	return conf, err
}
