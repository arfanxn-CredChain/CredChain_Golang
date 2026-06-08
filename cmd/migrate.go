package cmd

import (
	"context"
	"fmt"

	"CredChain_Golang/config"
	infraJobs "CredChain_Golang/infrastructure/jobs"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
	m, err := migrate.New(
		"file://infrastructure/database/migrations",
		*cfg.PostgresDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.Info("successfully ran database migrations up")

	if err := infraJobs.MigrateRiver(context.Background(), cfg); err != nil {
		return fmt.Errorf("river migrate up: %w", err)
	}
	logger.Info("successfully ran river migrations up")
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
	m, err := migrate.New(
		"file://infrastructure/database/migrations",
		*cfg.PostgresDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("rollback failed: %w", err)
	}

	logger.Info("successfully ran database migrations down")
	return nil
}
