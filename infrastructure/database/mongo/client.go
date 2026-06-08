package mongo

import (
	"context"
	"fmt"

	"CredChain_Golang/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
)

// NewClient connects to MongoDB and registers a lifecycle hook to disconnect.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*mongo.Client, error) {
	if cfg.MongoURI == nil || *cfg.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI is required")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(*cfg.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Ping(ctx, nil); err != nil {
				return fmt.Errorf("failed to ping mongo: %w", err)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error { return client.Disconnect(ctx) },
	})
	return client, nil
}

// NewDatabase returns the configured Mongo database handle. The default name
// lives in config.go (MongoDatabase); this provider only dereferences it.
func NewDatabase(client *mongo.Client, cfg *config.Config) *mongo.Database {
	return client.Database(*cfg.MongoDatabase)
}
