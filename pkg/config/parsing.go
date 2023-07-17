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

	for wf_name, wf := range sc.Workflows {
		for cb_name, cb := range wf.Callbacks {

			// Check for conflicts in on/not on conditions

			on := cb.On
			notOn := cb.NotOn
			for _, v := range on {
				if utils.Contains(notOn, v) {
					err := fmt.Errorf("the workflow status value '%v' for callback '%s' in workflow '%s' is conflicting and appears in both the on/ notOn clauses in the config", v, cb_name, wf_name)
					fmt.Println(err)
				}
			}

			callback_params := cb.Parameters
			handler_name := cb.Handler
			handler_params := sc.Handlers[handler_name].Parameters
			var callback_param_names []string
			var handler_param_names []string

			for hand_name, hand_param := range handler_params {
				if hand_param.IsRequired() {
					handler_param_names = append(handler_param_names, hand_name)
				}
			}

			for cb_name := range callback_params {
				callback_param_names = append(callback_param_names, cb_name)
			}

			// Check if all required handler parameters are defined in callback
			for _, v := range handler_param_names {
				if !utils.Contains(callback_param_names, v) {
					return fmt.Errorf("required handler parameter '%s' is not provided as a callback parameter for callback '%s' in workflow '%s' ", v, cb_name, wf_name)
				}
			}

			// Check if no additional callback parameters are defined that are not also required handler parameters
			if len(callback_param_names) != len(handler_param_names) {
				return fmt.Errorf("the number of callback parameters (%d) is not equal to the number of required handler parameters (%d) for callback %s in workflow '%s'", len(callback_param_names), len(handler_param_names), cb_name, wf_name)
			}

		}
	}

	return nil

}

func (p *HandlerParameter) IsRequired() bool {
	return !p.isDefaultSet
}

func (p *HandlerParameter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var iface map[string]interface{}
	var ok bool
	err := unmarshal(&iface)
	fmt.Println(err)
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

func main() {
	Parse("./fixtures/swoop-config.yml")

}
