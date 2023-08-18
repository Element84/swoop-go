package config

// TODO: how does this even work?
//
//	I think we need another type to actually fetch/retrive the value?
//	Or maybe it just checks the value at read time,
//	and if enough time has passed since last read it tries to read again?
//	If error on read after startup use last value and log error
//	-> next read should be scheduled, and we can use that to backoff reads on error
type HandlerSecret struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	TTL  int    `yaml:"ttl"`
}

type Handler struct {
	Name string      `yaml:"-"`
	Type HandlerType `yaml:"type"`
	// TODO: need a backoff type
	Backoff    map[string]int     `yaml:"backoff"`
	Parameters *HandlerParameters `yaml:"parameters"`
	Secrets    []*HandlerSecret   `yaml:"secrets"`

	HttpRequestConf *HttpRequest `yaml:"request"`
	// TODO: k8s options
	// just kubeconfig path for now?

	// TODO: cirrus options
	// not sure how this is going to work yet, just make a placeholder
}

func (h *Handler) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type p Handler

	err := unmarshal((*p)(h))
	if err != nil {
		return err
	}

	// TODO: Validate to ensure we have what we need/is allowed
	// Probably should start a convention here to use separate method for validation?
	// Consider https://github.com/dealancer/validate (but looks unmaintained...)

	return nil
}
