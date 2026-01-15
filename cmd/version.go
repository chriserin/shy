package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the current version of shy
const ShyVersion = "0.1.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of shy",
	Long:  "Print the version number of shy",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "shy version %s\n", ShyVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
