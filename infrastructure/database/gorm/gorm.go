package gorm

import (
	"context"

	"CredChain_Golang/config"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type GormParams struct {
	fx.In
	Config    *config.Config
	Lifecycle fx.Lifecycle
}

func NewGorm(p GormParams) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(*p.Config.PostgresDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(*p.Config.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(*p.Config.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(*p.Config.DBConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(*p.Config.DBConnMaxIdleTime) * time.Minute)

	p.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return sqlDB.Close()
		},
	})

	return db, nil
}
