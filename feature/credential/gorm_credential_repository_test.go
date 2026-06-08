package credential

import (
	"context"
	"strconv"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/testutil/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openCredRepo(t *testing.T) *gormCredentialRepository {
	t.Helper()
	return &gormCredentialRepository{db: db.OpenInMemorySQLite(t)}
}

func TestGormCredentialStore_FindByIds(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "b", FileHash: "0xbb"},
	)
	require.NoError(t, err)

	both, err := repo.FindByIds(ctx, "c1", "c2")
	require.NoError(t, err)
	assert.Len(t, both, 2)

	one, err := repo.FindByIds(ctx, "c1", "missing")
	require.NoError(t, err)
	assert.Len(t, one, 1)
	assert.Equal(t, "c1", one[0].ID)
}

func TestGormCredentialFindByFileHashes(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
	)
	require.NoError(t, err)

	found, err := repo.FindByFileHashes(ctx, "0xaa", "0xzz")
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "0xaa", found[0].FileHash)
}

func TestGormCredentialFindByHolderId(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "b", FileHash: "0xbb"},
	)
	require.NoError(t, err)

	found, err := repo.FindByHolderId(ctx, "h1")
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "c1", found[0].ID)
}

func TestGormCredentialFind(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
	)
	require.NoError(t, err)

	got, err := repo.Find(ctx, "c1", &domainQuery.Query{})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "c1", got.ID)

	missing, err := repo.Find(ctx, "missing", &domainQuery.Query{})
	assert.Error(t, err)
	assert.Nil(t, missing)
}

func TestGormCredentialUpdate(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
	)
	require.NoError(t, err)

	revokedAt := time.Now().UTC().Truncate(time.Second)
	updated, err := repo.Update(ctx,
		domain.Credential{ID: "c1", Name: "renamed", RevokedAt: &revokedAt},
	)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, "renamed", updated[0].Name)
	require.NotNil(t, updated[0].RevokedAt)
	assert.WithinDuration(t, revokedAt, *updated[0].RevokedAt, time.Second)
}

func TestGormCredentialGet(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "c" + strconv.Itoa(i+1)
		_, err := repo.Store(ctx, domain.Credential{ID: id, HolderUserID: "h1", IssuerUserID: "iss", Name: id, FileHash: "0x" + id})
		require.NoError(t, err)
	}

	creds, total, err := repo.Get(ctx, &domainQuery.Query{})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, creds, 5)
}
