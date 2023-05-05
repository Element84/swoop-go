package config

import (
	"fmt"
)

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Pass     string
	Name     string
	UrlExtra string
}

// TODO support parameter overrides for testing, maybe
func (conf *DatabaseConfig) Url() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s%s",
		conf.User,
		conf.Pass,
		conf.Host,
		conf.Port,
		conf.Name,
		conf.UrlExtra,
	)
}
