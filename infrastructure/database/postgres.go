package database

import (
	"time"

	"CredChain_Golang/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DB Wrapper around sqlx.DB
type DB struct {
	*sqlx.DB
}

type PostgresParams struct {
	fx.In
	Config *config.Config
}

// ConnectPostgres establishes a connection to the PostgreSQL database
func ConnectPostgres(p PostgresParams) (*DB, error) {
	// 1. Connect using sqlx
	db, err := sqlx.Connect("postgres", p.Config.PostgresDSN)
	if err != nil {
		return nil, err
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{db}, nil
}

// RunMigrationsUp applies upward migrations located in the local directory
func RunMigrationsUp(cfg *config.Config, logger *zap.Logger) error {
	m, err := migrate.New("file://infrastructure/database/migrations", cfg.PostgresDSN)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	logger.Info("database upward migrations applied successfully")
	return nil
}

// RunMigrationsDown rolls back all schema migrations located in the local directory
func RunMigrationsDown(cfg *config.Config, logger *zap.Logger) error {
	m, err := migrate.New("file://infrastructure/database/migrations", cfg.PostgresDSN)
	if err != nil {
		return err
	}

	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	logger.Info("database downward migrations applied successfully")
	return nil
}
