package cmd

import (
	"log"

	"github.com/element84/swoop-go/pkg/caboose/argo"

	"github.com/element84/swoop-go/pkg/cmdutil"
	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func init() {
	rootCmd.AddCommand(mkCabooseCmd())
}

func mkCabooseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "caboose",
		Short: "swoop-caboose commands for state updates",
	}
	s3Driver := &s3.S3Driver{}
	s3Driver.AddFlags(cmd.PersistentFlags())
	conf := &config.ConfigFile{}
	conf.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(func() *cobra.Command {
		configFlags := genericclioptions.NewConfigFlags(true)
		cmd := &cobra.Command{
			Use:   "argo",
			Short: "Run the caboose service for argo workflow integrations",
			Run: func(cmd *cobra.Command, args []string) {
				sc, err := conf.Parse()
				if err != nil {
					log.Fatal(err)
				}
				err = cmdutil.Run(
					"swoop-caboose",
					&argo.ArgoCaboose{
						S3Driver:       s3Driver,
						SwoopConfig:    sc,
						K8sConfigFlags: configFlags,
						DbConfig:       &db.PoolConfig{},
					},
				)
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
