package cmd

import (
	"context"

	"CredChain_Golang/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "credchain",
	Short: "CredChain Orchestration Backend",
	Long:  "CredChain CLI for managing the backend orchestration servers and administrative scripts.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		envFile, _ := cmd.Flags().GetString("env")
		cfg, err := config.NewConfig(envFile)
		if err != nil {
			return err
		}
		cmd.SetContext(context.WithValue(cmd.Context(), ConfigContextKey, cfg))
		return nil
	},
}

// Execute kicks off the Root Cobra Command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("env", "e", ".env", "Environment file to use")
}
