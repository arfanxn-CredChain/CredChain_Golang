package cmd

import (
	"context"
	"fmt"

	"CredChain_Golang/config"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	migrateMongoCmd.AddCommand(migrateMongoUpCmd)
	migrateMongoCmd.AddCommand(migrateMongoDownCmd)
	rootCmd.AddCommand(migrateMongoCmd)
}

var migrateMongoCmd = &cobra.Command{
	Use:   "migrate-mongo",
	Short: "MongoDB collection + index migration tools",
}

var migrateMongoUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Creates Mongo collections and indexes (idempotent)",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			infraLogger.Module,
			fx.Provide(NewConfigFromCmd(cmd)),
			fx.Invoke(migrateMongoUp),
		).Run()
	},
}

func migrateMongoConnect(cfg *config.Config) (*mongo.Database, func(context.Context) error, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(*cfg.MongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("connect mongo: %w", err)
	}
	return client.Database(*cfg.MongoDatabase), client.Disconnect, nil
}

func migrateMongoUp(cfg *config.Config, logger *zap.Logger) error {
	ctx := context.Background()
	db, disconnect, err := migrateMongoConnect(cfg)
	if err != nil {
		return err
	}
	defer disconnect(ctx)

	ttlSeconds := int32(*cfg.AIVerificationCacheTTLHours * 3600)

	if _, err := db.Collection("credential_extractions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "credential_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "file_hash", Value: 1}}},
		{Keys: bson.D{{Key: "ids.value", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("create extraction indexes: %w", err)
	}
	if _, err := db.Collection("credential_verifications").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "uploaded_file_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "created_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(ttlSeconds)},
	}); err != nil {
		return fmt.Errorf("create verification indexes: %w", err)
	}
	logger.Info("successfully ran mongo migrations up")
	return nil
}

var migrateMongoDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Drops Mongo collections (destructive)",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			infraLogger.Module,
			fx.Provide(NewConfigFromCmd(cmd)),
			fx.Invoke(migrateMongoDown),
		).Run()
	},
}

func migrateMongoDown(cfg *config.Config, logger *zap.Logger) error {
	ctx := context.Background()
	db, disconnect, err := migrateMongoConnect(cfg)
	if err != nil {
		return err
	}
	defer disconnect(ctx)
	for _, name := range []string{"credential_extractions", "credential_verifications"} {
		if err := db.Collection(name).Drop(ctx); err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
	}
	logger.Info("successfully ran mongo migrations down")
	return nil
}
