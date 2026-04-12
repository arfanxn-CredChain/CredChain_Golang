package database

import (
	"context"
	"fmt"

	"CredChain_Golang/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
)

// MongoDB wraps the mongo driver
type MongoDB struct {
	Client *mongo.Client
	Db     *mongo.Database
}

type MongoParams struct {
	fx.In
	Config *config.Config
}

// ConnectMongo evaluates uri and returns client
func ConnectMongo(p MongoParams) (*MongoDB, error) {
	if p.Config.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI not provided")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(p.Config.MongoURI))
	if err != nil {
		return nil, err
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	db := client.Database("credchain")
	return &MongoDB{Client: client, Db: db}, nil
}
