package jobs

import (
	"context"
	"time"

	"CredChain_Golang/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// MigrateRiver runs River's own table migrations (idempotent).
// Called from `make migrate-up` alongside the Postgres migrations.
func MigrateRiver(ctx context.Context, cfg *config.Config) error {
	pool, err := pgxpool.New(ctx, *cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer pool.Close()
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}

// RiverEnqueuer implements the Enqueuer interface by inserting River jobs.
type RiverEnqueuer struct {
	client *river.Client[pgx.Tx]
}

func (e *RiverEnqueuer) EnqueueExtract(ctx context.Context, args CredentialExtractArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}

// NewRiverClient creates a pgx pool + River client with the extraction worker,
// and registers lifecycle start/stop. Returns the client and an Enqueuer.
func NewRiverClient(lc fx.Lifecycle, cfg *config.Config, worker *CredentialExtractWorker, logger *zap.Logger) (Enqueuer, error) {
	pool, err := pgxpool.New(context.Background(), *cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: *cfg.RiverMaxWorkers}},
		Workers: workers,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				// Short delay to let FX app boot fully
				time.Sleep(1 * time.Second)
				if startErr := client.Start(ctx); startErr != nil {
					logger.Error("failed to start river client", zap.Error(startErr))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			_ = client.Stop(ctx)
			pool.Close()
			return nil
		},
	})
	return &RiverEnqueuer{client: client}, nil
}
