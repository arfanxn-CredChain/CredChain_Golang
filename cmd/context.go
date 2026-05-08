package cmd

import (
	"CredChain_Golang/config"

	"github.com/spf13/cobra"
)

type contextKey string

const ConfigContextKey contextKey = "config"

// ConfigFromCmd extracts *config.Config from a Cobra command's context.
// Panics if config is not set (should never happen if PersistentPreRunE ran).
func ConfigFromCmd(cmd *cobra.Command) *config.Config {
	return cmd.Context().Value(ConfigContextKey).(*config.Config)
}

// ConfigProviderFromCmd returns an FX provider function that supplies *config.Config.
// Use this in fx.Provide() blocks instead of inline closures.
func ConfigProviderFromCmd(cmd *cobra.Command) func() *config.Config {
	return func() *config.Config {
		return ConfigFromCmd(cmd)
	}
}
