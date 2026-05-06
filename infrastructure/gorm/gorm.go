package gorm

import (
	"CredChain_Golang/config"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type GormParams struct {
	fx.In
	Config *config.Config
}

func NewGorm(p GormParams) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(p.Config.PostgresDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(p.Config.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(p.Config.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(p.Config.DBConnMaxLifetime) * time.Minute)

	return db, nil
}
