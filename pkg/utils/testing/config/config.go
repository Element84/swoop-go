package config

import (
	"testing"

	"github.com/element84/swoop-go/pkg/config"
	test "github.com/element84/swoop-go/pkg/utils/testing"
)

func LoadConfigFixture(t *testing.T) *config.SwoopConfig {
	conf, err := config.Parse(test.GetFixture(t, "swoop-config.yml"))
	if err != nil {
		t.Fatalf("failed to parse config file: %s", err)
	}
	return conf
}
