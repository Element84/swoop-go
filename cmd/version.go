package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the swoop cli",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("STAC Workflow Open Orchestration Framework v0.0.1")
	},
}
