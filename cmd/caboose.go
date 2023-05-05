package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/element84/swoop-go/pkg/caboose"
	"github.com/element84/swoop-go/pkg/cmdutil"
)

func init() {
	rootCmd.AddCommand(mkCabooseCmd())
}

func mkCabooseCmd() *cobra.Command {
	var (
		config string
	)

	cmd := &cobra.Command{
		Use:   "caboose",
		Short: "swoop-caboose commands for state updates",
	}
	databaseConfig := databaseFlags(cmd.PersistentFlags())
	// TODO: convert config flag to be like others
	cmd.PersistentFlags().StringVarP(
		&config, "config-file", "f", "", "swoop-caboose config file path (required; SWOOP_CONFIG_FILE)",
	)
	cmd.MarkPersistentFlagRequired("config-file")

	cmd.AddCommand(func() *cobra.Command {
		configFlags := genericclioptions.NewConfigFlags(true)
		cmd := &cobra.Command{
			Use:   "argo",
			Short: "Run the caboose service for argo workflow integrations",
			Run: func(cmd *cobra.Command, args []string) {
				err := cmdutil.Run(&caboose.ArgoCaboose{
					ConfigFile:     config,
					DatabaseConfig: databaseConfig,
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
