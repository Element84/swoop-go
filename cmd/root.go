package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:   "swoop",
		Short: "STAC Workflow Open Orchestration Framework - go utilities",
		Long: `The STAC Workflow Open Orchestration Framework (swoop) is
a collection of services to implement a geospatial processing pipeline.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig(cmd)
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func initConfig(cmd *cobra.Command) {
	v := viper.New()
	v.SetEnvPrefix("swoop")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	bindFlags(cmd.Flags(), v)
}
