package jobs

import (
	"context"
	"fmt"

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
		Queues:       map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: *cfg.RiverMaxWorkers}},
		Workers:      workers,
		ErrorHandler: worker,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}
	started := false
	lc.Append(fx.Hook{
		// river.Client.Start is non-blocking: it launches the worker pool in
		// the background and returns immediately. Call it synchronously so the
		// "started" flag is set before OnStop can run (no Start/Stop race).
		OnStart: func(ctx context.Context) error {
			if startErr := client.Start(ctx); startErr != nil {
				pool.Close()
				return fmt.Errorf("start river client: %w", startErr)
			}
			started = true
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if started {
				if stopErr := client.Stop(ctx); stopErr != nil {
					logger.Error("failed to stop river client", zap.Error(stopErr))
				}
			}
			pool.Close()
			return nil
		},
	})
	return &RiverEnqueuer{client: client}, nil
}
