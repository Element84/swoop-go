package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/element84/swoop-go/pkg/config"
)

func init() {
	rootCmd.AddCommand(mkConfigCmd())
}

func mkConfigCmd() *cobra.Command {
	var (
		configFile string
	)

	cmd := &cobra.Command{
		Use:   "config",
		Short: "swoop command to verify/dump parsed config",
		Run: func(cmd *cobra.Command, args []string) {
			swoopConf, err := config.Parse(configFile)
			if err != nil {
				log.Fatalf("error: %v", err)
			}

			d, err := yaml.Marshal(swoopConf)
			if err != nil {
				log.Fatalf("error: %v", err)
			}

			fmt.Printf("%s\n", string(d))
		},
	}
	cmd.PersistentFlags().StringVarP(
		&configFile, "config-file", "f", "", "swoop config file path (required; SWOOP_CONFIG_FILE)",
	)
	cmd.MarkPersistentFlagRequired("config-file")

	return cmd
}
