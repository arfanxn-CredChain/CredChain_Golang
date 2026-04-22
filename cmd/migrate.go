package cmd

import (
	"fmt"

	"CredChain_Golang/config"
	"CredChain_Golang/infrastructure/database"

	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.NewConfig(".env")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		logger, _ := zap.NewProduction()
		defer logger.Sync()

		err = database.MigrateUp(cfg, logger)
		if err != nil {
			logger.Error("migration failed", zap.Error(err))
			return err
		}

		logger.Info("successfully ran database migrations up")
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Reverts the schema downwards",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.NewConfig(".env")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		logger, _ := zap.NewProduction()
		defer logger.Sync()

		err = database.MigrateDown(cfg, logger)
		if err != nil {
			logger.Error("rollback failed", zap.Error(err))
			return err
		}

		logger.Info("successfully ran database migrations down")
		return nil
	},
}
