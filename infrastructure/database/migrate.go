package database

import (
	"fmt"

	"CredChain_Golang/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

// MigrateUp runs all pending upward migrations
func MigrateUp(cfg *config.Config, logger *zap.Logger) error {
	m, err := migrate.New(
		"file://infrastructure/database/migrations",
		cfg.PostgresDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// MigrateDown rolls back one migration step
func MigrateDown(cfg *config.Config, logger *zap.Logger) error {
	m, err := migrate.New(
		"file://infrastructure/database/migrations",
		cfg.PostgresDSN,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}
