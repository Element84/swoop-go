// package config
package config

import (
	"fmt"
	"io/ioutil"

	"github.com/element84/swoop-go/pkg/utils"

	"gopkg.in/yaml.v3"
)

type SwoopConfig struct {
	Workflows  map[string]Workflow  `yaml:"workflows"`
	Handlers   map[string]Handler   `yaml:"handlers"`
	Conductors map[string]Conductor `yaml:"conductors"`
}

// Callback structs
type CallbackParameter struct {
	Value string `yaml:"value"`
}

type Callback struct {
	Handler       string                       `yaml:"handler"`
	Type          string                       `yaml:"type"`
	FeatureFilter string                       `yaml:"feature_filter,omitempty"`
	On            []string                     `yaml:"on,omitempty"`
	NotOn         []string                     `yaml:"notOn,omitempty"`
	Enabled       bool                         `yaml:"enabled,omitempty"`
	Parameters    map[string]CallbackParameter `yaml:"parameters,omitempty"`
}

type Workflow struct {
	Callbacks map[string]Callback `yaml:"callbacks"`
}

// Handler structs
type Handler struct {
	Type        string                      `yaml:"type,omitempty"`
	Url         string                      `yaml:"url,omitempty"`
	RequestBody string                      `yaml:"requestBody,omitempty"`
	Operation   string                      `yaml:"operation,omitempty"`
	Secrets     []Secret                    `yaml:"secrets,omitempty"`
	Headers     map[string]string           `yaml:"headers,omitempty"`
	Backoff     map[string]int              `yaml:"backoff,omitempty"`
	Errors      []Error                     `yaml:"errors,omitempty"`
	Parameters  map[string]HandlerParameter `yaml:"parameters,omitempty"`
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

type HandlerParameter struct {
	isDefaultSet bool
	Default      interface{} `yaml:"default,omitempty"`
}

// Conductor structs

type Conductor struct {
	Handlers []string `yaml:"handlers,omitempty"`
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

func ValidateConfig(sc *SwoopConfig) error {
	for wfName, wf := range sc.Workflows {
		for cbName, cb := range wf.Callbacks {
			var (
				callbackParamNames []string
				handlerParamNames  []string
				requiredParamNames []string
			)

			// Check for conflicts in on/not on conditions
			on := cb.On
			notOn := cb.NotOn
			for _, v := range on {
				if utils.Contains(notOn, v) {
					return fmt.Errorf(
						"the workflow status value '%v' for callback '%s' in workflow '%s' is conflicting and appears in both the on/ notOn clauses in the config",
						v,
						cbName,
						wfName,
					)
				}
			}

			callbackParams := cb.Parameters
			handlerName := cb.Handler

			for name, param := range sc.Handlers[handlerName].Parameters {
				handlerParamNames = append(handlerParamNames, name)
				if param.IsRequired() {
					requiredParamNames = append(requiredParamNames, name)
				}
			}

			for name := range callbackParams {
				callbackParamNames = append(callbackParamNames, name)
			}

			// Check if all required handler parameters are defined in callback
			for _, v := range requiredParamNames {
				if !utils.Contains(callbackParamNames, v) {
					return fmt.Errorf(
						"handler '%s' required parameter '%s' is not defined on callback '%s' in workflow '%s' ",
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
						"callback '%s' parameter '%s' is not a known parameter for handler '%s' in workflow '%s' ",
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

func (p *HandlerParameter) IsRequired() bool {
	return !p.isDefaultSet
}

func (p *HandlerParameter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var (
		iface map[string]interface{}
		ok    bool
	)

	err := unmarshal(&iface)
	if err != nil {
		return err
	}

	p.isDefaultSet = false
	p.Default, ok = iface["default"]
	if ok {
		p.isDefaultSet = true
	}

	return nil
}

func Parse(configFile string) (*SwoopConfig, error) {
	conf := &SwoopConfig{}

	err := loadYaml(configFile, conf)
	if err != nil {
		return nil, err
	}

	err = ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
