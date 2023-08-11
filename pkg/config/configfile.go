package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ConfigFile struct {
	path string
}

func (cf *ConfigFile) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(
		&cf.path,
		"config-file",
		"f",
		"",
		"swoop config file path (required; SWOOP_CONFIG_FILE)",
	)
	cobra.MarkFlagRequired(fs, "config-file")
}

func (cf *ConfigFile) Parse() (*SwoopConfig, error) {
	sc, err := Parse(cf.path)
	if err != nil {
		err = fmt.Errorf("error parsing config: %s", err)
	}
	return sc, err
}
