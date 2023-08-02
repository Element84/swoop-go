package config

import (
	"fmt"
)

type Callback struct {
	Name          string              `yaml:"-"`
	HandlerName   string              `yaml:"handler"`
	Type          CallbackType        `yaml:"type"`
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
