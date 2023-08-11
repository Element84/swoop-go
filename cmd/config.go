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
	conf := &config.ConfigFile{}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "swoop command to verify/dump parsed config",
		Run: func(cmd *cobra.Command, args []string) {
			sc, err := conf.Parse()
			if err != nil {
				log.Fatal(err)
			}

			d, err := yaml.Marshal(sc)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%s\n", string(d))
		},
	}
	conf.AddFlags(cmd.PersistentFlags())

	return cmd
}
