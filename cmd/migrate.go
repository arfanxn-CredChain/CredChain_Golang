package cmd

import (
	"CredChain_Golang/config"
	"CredChain_Golang/infrastructure/database"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database schema migration tools",
	Long:  "Contains subcommands to migrate the database schema upwards or downwards.",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Executes the upward migrations",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			infraLogger.Module,
			fx.Provide(NewConfigFromCmd(cmd)),
			fx.Invoke(migrateUp),
		).Run()
	},
}

func migrateUp(cfg *config.Config, logger *zap.Logger) error {
	err := database.MigrateUp(cfg, logger)
	if err != nil {
		logger.Error("migration failed", zap.Error(err))
		return err
	}

	logger.Info("successfully ran database migrations up")
	return nil
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Reverts the schema downwards",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			infraLogger.Module,
			fx.Provide(NewConfigFromCmd(cmd)),
			fx.Invoke(migrateDown),
		).Run()
	},
}

func migrateDown(cfg *config.Config, logger *zap.Logger) error {
	err := database.MigrateDown(cfg, logger)
	if err != nil {
		logger.Error("rollback failed", zap.Error(err))
		return err
	}

	logger.Info("successfully ran database migrations down")
	return nil
}
