package user

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/testutil/db"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/stretchr/testify/assert"
)

func newRepo(t *testing.T) domain.UserRepository {
	t.Helper()
	return NewGormUserRepository(db.OpenInMemorySQLite(t))
}

func TestGormUserRepository_Store_AutoGeneratesULID(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID(""), fixtures.WithEmail("auto@x.com"))
	stored, err := repo.Store(context.Background(), u)
	assert.NoError(t, err)
	assert.Len(t, stored, 1)
	assert.NotEmpty(t, stored[0].Id)
}

func TestGormUserRepository_Store_PreservesProvidedID(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("custom-id"), fixtures.WithEmail("custom@x.com"))
	stored, err := repo.Store(context.Background(), u)
	assert.NoError(t, err)
	assert.Equal(t, "custom-id", stored[0].Id)
}

func TestGormUserRepository_Store_Batch(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithEmail("a@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithEmail("b@x.com"))
	stored, err := repo.Store(context.Background(), u1, u2)
	assert.NoError(t, err)
	assert.Len(t, stored, 2)
}

func TestGormUserRepository_Find(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("findme"), fixtures.WithEmail("find@x.com"))
	_, _ = repo.Store(context.Background(), u)

	found, err := repo.Find(context.Background(), "findme")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "find@x.com", found.Email)
}

func TestGormUserRepository_Find_NotFound(t *testing.T) {
	repo := newRepo(t)
	_, err := repo.Find(context.Background(), "nope")
	assert.Error(t, err)
}

func TestGormUserRepository_FindByEmails_Empty(t *testing.T) {
	repo := newRepo(t)
	users, err := repo.FindByEmails(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, users)
}

func TestGormUserRepository_FindByEmails_NoMatches(t *testing.T) {
	repo := newRepo(t)
	users, err := repo.FindByEmails(context.Background(), "missing@x.com")
	assert.NoError(t, err)
	assert.Empty(t, users)
}

func TestGormUserRepository_FindByEmails_Multiple(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("b@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("c@x.com")),
	)
	users, err := repo.FindByEmails(context.Background(), "a@x.com", "c@x.com")
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestGormUserRepository_FindByRole(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("a@x.com"), fixtures.WithRole(domain.RoleAdmin)),
		fixtures.NewDomainUser(fixtures.WithEmail("b@x.com"), fixtures.WithRole(domain.RoleHolder)),
	)
	admins, err := repo.FindByRole(context.Background(), domain.RoleAdmin)
	assert.NoError(t, err)
	assert.Len(t, admins, 1)
	assert.Equal(t, "a@x.com", admins[0].Email)
}

func TestGormUserRepository_FindByIds_Empty(t *testing.T) {
	repo := newRepo(t)
	got, err := repo.FindByIds(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestGormUserRepository_FindByIds_Multiple(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithID("id1"), fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithID("id2"), fixtures.WithEmail("b@x.com")),
	)
	got, err := repo.FindByIds(context.Background(), "id1", "id2", "missing")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestGormUserRepository_Update(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("upd"), fixtures.WithEmail("old@x.com"))
	_, _ = repo.Store(context.Background(), u)

	u.Email = "new@x.com"
	updated, err := repo.Update(context.Background(), u)
	assert.NoError(t, err)
	assert.Equal(t, "new@x.com", updated.Email)
}

func TestGormUserRepository_Destroy_Empty(t *testing.T) {
	repo := newRepo(t)
	n, err := repo.Destroy(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, n)
}

func TestGormUserRepository_Destroy(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("d1"), fixtures.WithEmail("a@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("d2"), fixtures.WithEmail("b@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2)

	n, err := repo.Destroy(context.Background(), "d1", "d2")
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)

	_, findErr := repo.Find(context.Background(), "d1")
	assert.Error(t, findErr)
}

func TestGormUserRepository_UpdateRole_BatchCASE(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithEmail("a@x.com"), fixtures.WithRole(domain.RoleHolder))
	u2 := fixtures.NewDomainUser(fixtures.WithID("u2"), fixtures.WithEmail("b@x.com"), fixtures.WithRole(domain.RoleHolder))
	_, _ = repo.Store(context.Background(), u1, u2)

	u1.Role = domain.RoleIssuer
	u2.Role = domain.RoleAdmin
	updated, n, err := repo.UpdateRole(context.Background(), u1, u2)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, n)
	assert.Len(t, updated, 2)
	roles := map[string]domain.Role{updated[0].Id: updated[0].Role, updated[1].Id: updated[1].Role}
	assert.Equal(t, domain.RoleIssuer, roles["u1"])
	assert.Equal(t, domain.RoleAdmin, roles["u2"])
}

func TestGormUserRepository_UpdateRole_Empty(t *testing.T) {
	repo := newRepo(t)
	got, n, err := repo.UpdateRole(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, got)
	assert.EqualValues(t, 0, n)
}

func TestGormUserRepository_Get_DefaultOrder(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("b@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, users, 2)
}

func TestGormUserRepository_Get_SearchByName(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithName("Alice"), fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithName("Bob"), fixtures.WithEmail("b@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10, Search: "ali"}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
}

func TestGormUserRepository_Get_SearchCaseInsensitive(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithEmail("alice@x.com")),
		fixtures.NewDomainUser(fixtures.WithEmail("bob@x.com")),
	)
	q := &domainQuery.Query{Page: 1, Limit: 10, Search: "ALICE"}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 1, "search must be case-insensitive")
}

func TestGormUserRepository_Get_PageLimit(t *testing.T) {
	repo := newRepo(t)
	for i := 0; i < 5; i++ {
		_, _ = repo.Store(context.Background(), fixtures.NewDomainUser())
	}
	q := &domainQuery.Query{Page: 1, Limit: 2}
	users, total, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, users, 2)
}

func TestGormUserRepository_Get_SortByName(t *testing.T) {
	repo := newRepo(t)
	_, _ = repo.Store(context.Background(),
		fixtures.NewDomainUser(fixtures.WithName("Charlie"), fixtures.WithEmail("c@x.com")),
		fixtures.NewDomainUser(fixtures.WithName("Alice"), fixtures.WithEmail("a@x.com")),
		fixtures.NewDomainUser(fixtures.WithName("Bob"), fixtures.WithEmail("b@x.com")),
	)
	q := &domainQuery.Query{
		Page: 1, Limit: 10,
		Sorts: []domainQuery.Sort{{Column: "name", Order: domainQuery.SortAsc}},
	}
	users, _, err := repo.Get(context.Background(), q)
	assert.NoError(t, err)
	assert.Len(t, users, 3)
	assert.Equal(t, "Alice", *users[0].Name)
	assert.Equal(t, "Bob", *users[1].Name)
	assert.Equal(t, "Charlie", *users[2].Name)
}
