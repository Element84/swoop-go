package cmd

import (
	"log"

	"github.com/element84/swoop-go/pkg/caboose"

	"github.com/element84/swoop-go/pkg/cmdutil"
	"github.com/element84/swoop-go/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"github.com/element84/swoop-go/pkg/s3"
)

func init() {
	rootCmd.AddCommand(mkCabooseCmd())
}

func mkCabooseCmd() *cobra.Command {
	var (
		configFile string
	)

	cmd := &cobra.Command{
		Use:   "caboose",
		Short: "swoop-caboose commands for state updates",
	}
	s3Config := &s3.S3Driver{}
	s3Config.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().StringVarP(
		&configFile, "config-file", "f", "", "swoop-caboose config file path (required; SWOOP_CONFIG_FILE)",
	)
	cmd.MarkPersistentFlagRequired("config-file")

	cmd.AddCommand(func() *cobra.Command {
		configFlags := genericclioptions.NewConfigFlags(true)
		cmd := &cobra.Command{
			Use:   "argo",
			Short: "Run the caboose service for argo workflow integrations",
			Run: func(cmd *cobra.Command, args []string) {
				sc, err := config.Parse(configFile)
				if err != nil {
					log.Fatalf("Error parsing config: %s", err)

				}
				err = cmdutil.Run(&caboose.ArgoCaboose{
					SwoopConfig:    sc,
					K8sConfigFlags: configFlags,
				})
				if err != nil {
					log.Fatalf("Error in caboose: %s", err)

				}
			},
		}
		configFlags.AddFlags(cmd.Flags())
		return cmd
	}())

	return cmd
}
