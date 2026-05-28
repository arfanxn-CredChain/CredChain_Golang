package user

import (
	"context"
	"testing"
	"time"

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
	assert.Len(t, updated, 1)
	assert.Equal(t, "new@x.com", updated[0].Email)
}

func TestGormUserRepository_Delete_Empty(t *testing.T) {
	repo := newRepo(t)
	n, err := repo.Delete(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, n)
}

func TestGormUserRepository_Delete(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("d1"), fixtures.WithEmail("a@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("d2"), fixtures.WithEmail("b@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2)

	n, err := repo.Delete(context.Background(), "d1", "d2")
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

func TestGormUserRepository_Delete_SoftDelete_HidesFromFind(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd1"), fixtures.WithEmail("sd1@x.com"))
	_, _ = repo.Store(context.Background(), u)
	_, err := repo.Delete(context.Background(), "sd1")
	assert.NoError(t, err)
	_, findErr := repo.Find(context.Background(), "sd1")
	assert.Error(t, findErr, "soft-deleted user must not be returned by Find")
}

func TestGormUserRepository_Delete_SoftDelete_HidesFromGet(t *testing.T) {
	repo := newRepo(t)
	u1 := fixtures.NewDomainUser(fixtures.WithID("sd2"), fixtures.WithEmail("sd2@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("sd3"), fixtures.WithEmail("sd3@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2)
	_, _ = repo.Delete(context.Background(), "sd2")
	users, total, err := repo.Get(context.Background(), &domainQuery.Query{})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, users, 1)
	assert.Equal(t, "sd3", users[0].Id)
}

func TestGormUserRepository_Delete_SoftDelete_HidesFromFindByIds(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd4"), fixtures.WithEmail("sd4@x.com"))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "sd4")
	found, err := repo.FindByIds(context.Background(), "sd4")
	assert.NoError(t, err)
	assert.Empty(t, found)
}

func TestGormUserRepository_Delete_SoftDelete_HidesFromFindByEmails(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd5"), fixtures.WithEmail("sd5@x.com"))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "sd5")
	found, err := repo.FindByEmails(context.Background(), "sd5@x.com")
	assert.NoError(t, err)
	assert.Empty(t, found)
}

func TestGormUserRepository_Delete_SoftDelete_HidesFromFindByRole(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("sd6"), fixtures.WithEmail("sd6@x.com"), fixtures.WithRole(domain.RoleHolder))
	_, _ = repo.Store(context.Background(), u)
	_, _ = repo.Delete(context.Background(), "sd6")
	found, err := repo.FindByRole(context.Background(), domain.RoleHolder)
	assert.NoError(t, err)
	assert.Empty(t, found)
}

func TestGormUserRepository_Update_BirthDateSet_PersistsValue(t *testing.T) {
	repo := newRepo(t)
	u := fixtures.NewDomainUser(fixtures.WithID("bd1"), fixtures.WithEmail("bd1@x.com"))
	_, _ = repo.Store(context.Background(), u)
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := repo.Update(context.Background(), domain.User{Id: "bd1", BirthDate: &bd})
	assert.NoError(t, err)
	got, _ := repo.Find(context.Background(), "bd1")
	assert.NotNil(t, got.BirthDate)
	assert.Equal(t, 1990, got.BirthDate.Year())
}

func TestGormUserRepository_Update_BirthDateNil_PreservesExisting(t *testing.T) {
	repo := newRepo(t)
	bd := time.Date(2000, 5, 15, 0, 0, 0, 0, time.UTC)
	u := fixtures.NewDomainUser(fixtures.WithID("bd2"), fixtures.WithEmail("bd2@x.com"))
	u.BirthDate = &bd
	_, _ = repo.Store(context.Background(), u)
	name := "Renamed"
	_, err := repo.Update(context.Background(), domain.User{Id: "bd2", Name: &name})
	assert.NoError(t, err)
	got, _ := repo.Find(context.Background(), "bd2")
	assert.NotNil(t, got.BirthDate, "nil pointer in update payload must not clear existing date")
	assert.Equal(t, 2000, got.BirthDate.Year())
}

func TestGormUserRepository_Update_BirthDateExplicitClear_SetsNull(t *testing.T) {
	t.Skip("explicit-clear-to-null is intentionally unsupported with struct-based Updates; revisit if product requires it")
}

func TestGormUserRepository_Update_BatchCASE_MixedColumns(t *testing.T) {
	repo := newRepo(t)

	u1 := fixtures.NewDomainUser(fixtures.WithID("bc1"), fixtures.WithEmail("bc1@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithID("bc2"), fixtures.WithEmail("bc2@x.com"))
	u3 := fixtures.NewDomainUser(fixtures.WithID("bc3"), fixtures.WithEmail("bc3@x.com"))
	_, _ = repo.Store(context.Background(), u1, u2, u3)

	name1 := "Alice"
	num2 := "99999"
	phone3 := "+6281234567890"

	updated, err := repo.Update(context.Background(),
		domain.User{Id: "bc1", Name: &name1},
		domain.User{Id: "bc2", Number: &num2},
		domain.User{Id: "bc3", PhoneNumber: &phone3},
	)
	assert.NoError(t, err)
	assert.Len(t, updated, 3)

	byID := make(map[string]domain.User)
	for _, u := range updated {
		byID[u.Id] = u
	}

	assert.Equal(t, "Alice", *byID["bc1"].Name)
	assert.Equal(t, "bc1@x.com", byID["bc1"].Email)

	assert.Equal(t, "99999", *byID["bc2"].Number)
	assert.Equal(t, "bc2@x.com", byID["bc2"].Email)

	assert.Equal(t, "+6281234567890", *byID["bc3"].PhoneNumber)
	assert.Equal(t, "bc3@x.com", byID["bc3"].Email)

	assert.Nil(t, byID["bc1"].Number)
	assert.Nil(t, byID["bc2"].Name)
	assert.Nil(t, byID["bc3"].Name)
}
