package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "credchain",
	Short: "CredChain Orchestration Backend",
	Long:  "CredChain CLI for managing the backend orchestration servers and administrative scripts.",
}

// Execute kicks off the Root Cobra Command
func Execute() error {
	return rootCmd.Execute()
}
