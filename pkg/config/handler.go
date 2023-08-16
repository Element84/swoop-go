package config

type Handler struct {
	Name        string             `yaml:"-"`
	Type        HandlerType        `yaml:"type"`
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

type Response struct {
	Status    int    `yaml:"status,omitempty"`
	Message   string `yaml:"message,omitempty"`
	Retryable string `yaml:"retryable,omitempty"`
}
