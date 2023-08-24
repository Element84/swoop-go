package cmd

import (
	"log"

	"github.com/element84/swoop-go/pkg/conductor"

	"github.com/element84/swoop-go/pkg/cmdutil"
	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mkConductorCmd())
}

func mkConductorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conductor",
		Short: "swoop-conductor commands for action handling",
	}
	s3Driver := &s3.S3Driver{}
	s3Driver.AddFlags(cmd.PersistentFlags())
	conf := &config.ConfigFile{}
	conf.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(func() *cobra.Command {
		cmd := &cobra.Command{
			Use:   "run [instance name]",
			Short: "Run the conductor service to handle actions via pg notifications",
			Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
			Run: func(cmd *cobra.Command, args []string) {
				sc, err := conf.Parse()
				if err != nil {
					log.Fatal(err)
				}
				err = cmdutil.Run(
					"swoop-conductor",
					&conductor.PgConductor{
						InstanceName: args[0],
						S3Driver:     s3Driver,
						SwoopConfig:  sc,
						DbConfig:     &db.ConnectConfig{},
					},
				)
				if err != nil {
					log.Fatalf("Error in conductor: %s", err)
				}
			},
		}
		return cmd
	}())

	return cmd
}
