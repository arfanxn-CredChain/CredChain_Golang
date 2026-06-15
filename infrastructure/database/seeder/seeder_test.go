package seeder_test

import (
	"context"
	"errors"
	"testing"

	"CredChain_Golang/infrastructure/database/seeder"
	"github.com/stretchr/testify/assert"
)

type testSeeder struct {
	name string
	run  func() error
}

func (s *testSeeder) Name() string                   { return s.name }
func (s *testSeeder) Seed(ctx context.Context) error { return s.run() }

func TestRegistry_RunVariadicNames(t *testing.T) {
	ctx := context.Background()
	var userRan, credRan bool

	r := seeder.NewRegistry(
		&testSeeder{name: "user", run: func() error { userRan = true; return nil }},
		&testSeeder{name: "credential", run: func() error { credRan = true; return nil }},
	)

	err := r.Run(ctx, "user")
	assert.NoError(t, err)
	assert.True(t, userRan)
	assert.False(t, credRan)
}

func TestRegistry_RunAll_WhenNoNamesProvided(t *testing.T) {
	ctx := context.Background()
	var userRan, credRan bool

	r := seeder.NewRegistry(
		&testSeeder{name: "user", run: func() error { userRan = true; return nil }},
		&testSeeder{name: "credential", run: func() error { credRan = true; return nil }},
	)

	err := r.Run(ctx)
	assert.NoError(t, err)
	assert.True(t, userRan)
	assert.True(t, credRan)
}

func TestRegistry_RunMultipleNames(t *testing.T) {
	ctx := context.Background()
	var userRan, credRan bool

	r := seeder.NewRegistry(
		&testSeeder{name: "user", run: func() error { userRan = true; return nil }},
		&testSeeder{name: "credential", run: func() error { credRan = true; return nil }},
	)

	err := r.Run(ctx, "user", "credential")
	assert.NoError(t, err)
	assert.True(t, userRan)
	assert.True(t, credRan)
}

func TestRegistry_RunAllStopsOnFirstError(t *testing.T) {
	ctx := context.Background()
	var credRan bool

	r := seeder.NewRegistry(
		&testSeeder{name: "user", run: func() error { return errors.New("fail") }},
		&testSeeder{name: "credential", run: func() error { credRan = true; return nil }},
	)

	err := r.Run(ctx)
	assert.Error(t, err)
	assert.False(t, credRan)
}

func TestRegistry_RunNotFound(t *testing.T) {
	ctx := context.Background()
	r := seeder.NewRegistry()

	err := r.Run(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "seeder not found")
}
