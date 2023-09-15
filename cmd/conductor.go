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
		pgConductor := &conductor.PgConductor{
			DbConfig: &db.ConnectConfig{},
		}

		cmd := &cobra.Command{
			Use:   "run",
			Short: "Run the conductor service to handle actions via pg notifications",
			Run: func(cmd *cobra.Command, args []string) {
				sc, err := conf.Parse()
				if err != nil {
					log.Fatal(err)
				}
				pgConductor.SwoopConfig = sc
				pgConductor.S3 = s3.NewSwoopS3(s3.NewJsonClient(s3Driver))
				err = cmdutil.Run(
					"swoop-conductor",
					pgConductor,
				)
				if err != nil {
					log.Fatalf("Error in conductor: %s", err)
				}
			},
		}

		pgConductor.AddFlags(cmd.Flags())
		return cmd
	}())

	return cmd
}
