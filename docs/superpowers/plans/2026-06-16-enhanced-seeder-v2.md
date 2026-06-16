# Enhanced Seeder v2 — NIP/NIM, Meta, Soft-Delete, Gender Reorder

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance the user seeder with NIP/NIM employee/student numbers, random meta, soft-deleted users, and reorder the `gender` column after `email` in the DB schema and GORM model.

**Architecture:** The `000001_initial_schema` migration is edited in-place to swap `birth_date`/`gender` column order (no new migration file). The model struct swaps those two field positions. The user seeder is rewritten to construct `domain.User` directly with all fields populated, adds `seedGenerateNIP`/`seedGenerateNIM`/`seedRandomAlphaKey` helpers, and calls `repo.Delete()` for 5 soft-deleted users. The seed-chain command processes deleted users by setting `RoleNone` on-chain while preserving DB roles.

**Tech Stack:** Go 1.25.1, GORM, PostgreSQL, Cobra, Uber FX, go-ethereum, testify

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `infrastructure/database/migrations/000001_initial_schema.up.sql` | Modify | Swap `birth_date` / `gender` column order |
| `infrastructure/database/migrations/000001_initial_schema.down.sql` | Modify | Swap back (no-op since down drops table) |
| `infrastructure/database/gorm/model/user.go` | Modify | Swap `BirthDate` / `Gender` field positions, swap in `ToDomain`/`FromDomain` |
| `infrastructure/database/seeder/user_seeder.go` | Modify | Rewrite: `domain.User` directly, NIP/NIM, meta, soft-delete |
| `infrastructure/database/seeder/user_seeder_test.go` | Modify | Add tests for numbers, meta, deleted users |
| `cmd/seed_chain.go` | Modify | Deleted users → `RoleNone` on-chain, keep DB role |

---

## Task 1: Gender Column Reorder

**Files:**
- Modify: `infrastructure/database/migrations/000001_initial_schema.up.sql:19-21`
- Modify: `infrastructure/database/migrations/000001_initial_schema.down.sql:23-24` (reexport enums after drop)
- Modify: `infrastructure/database/gorm/model/user.go:15-17`

Swap `birth_date` and `gender` so that `email` is followed by `gender` then `birth_date`.

- [ ] **Step 1: Edit migration up.sql**

In `infrastructure/database/migrations/000001_initial_schema.up.sql`, lines 19-21 currently read:

```sql
    email VARCHAR(256) UNIQUE NOT NULL,
    birth_date DATE,
    gender gender,
```

Change to:

```sql
    email VARCHAR(256) UNIQUE NOT NULL,
    gender gender,
    birth_date DATE,
```

- [ ] **Step 2: Edit migration down.sql**

In `infrastructure/database/migrations/000001_initial_schema.down.sql`, after `DROP TABLE IF EXISTS users;` and before `DROP TYPE IF EXISTS role;`, the down migration already drops the whole table and types. No column reorder needed in the down file — the table is recreated from scratch by `up.sql` on next migrate-up. **No changes to down.sql needed.**

- [ ] **Step 3: Edit GORM model struct**

In `infrastructure/database/gorm/model/user.go`, swap the field positions of `BirthDate` and `Gender`. Currently lines 15-18:

```go
	Email                     string         `gorm:"type:varchar(255);uniqueIndex;column:email" json:"email"`
	BirthDate                 *time.Time     `gorm:"column:birth_date" json:"birth_date"`
	Gender                    *string        `gorm:"type:gender;column:gender" json:"gender"`
	Meta                      map[string]any `gorm:"type:jsonb;serializer:json;column:meta" json:"meta"`
```

Change to:

```go
	Email                     string         `gorm:"type:varchar(255);uniqueIndex;column:email" json:"email"`
	Gender                    *string        `gorm:"type:gender;column:gender" json:"gender"`
	BirthDate                 *time.Time     `gorm:"column:birth_date" json:"birth_date"`
	Meta                      map[string]any `gorm:"type:jsonb;serializer:json;column:meta" json:"meta"`
```

Also swap the field order in `ToDomain()` (lines 43-44) and `FromDomainUser()` (lines 71-72):

**ToDomain:** change from `BirthDate: m.BirthDate, Gender: gender,` to `Gender: gender, BirthDate: m.BirthDate,`

**FromDomainUser:** change from `BirthDate: u.BirthDate, Gender: gender,` to `Gender: gender, BirthDate: u.BirthDate,`

- [ ] **Step 4: Run tests to verify no regressions**

```bash
go test ./infrastructure/database/gorm/model/ -v
go test ./feature/user/ -v
```

Expected: all existing tests pass.

- [ ] **Step 5: Commit**

```bash
git add infrastructure/database/migrations/000001_initial_schema.up.sql \
        infrastructure/database/gorm/model/user.go
git commit -m "refactor: reorder gender column after email in schema and model"
```

---

## Task 2: Enhanced User Seeder

**Files:**
- Modify: `infrastructure/database/seeder/user_seeder.go`
- Modify: `infrastructure/database/seeder/user_seeder_test.go`

Full rewrite of seeder: construct `domain.User` directly (no `definedUser` helper struct). Add NIP (employee) for Issuer+ roles, NIM (student) for Holder roles. Add random meta `{"key":"<UPPERALPHA8>"}` for ~half the users. Soft-delete 5 users via `repo.Delete()`.

### NIP Format: `YYYYMMDDYYYYMMXNNN`

| Part | Length | Description |
|------|--------|-------------|
| YYYYMMDD | 8 | Date of Birth |
| YYYYMM | 6 | Year & Month of Recruitment (DOB + 21 years) |
| X | 1 | Gender: 1=Male, 2=Female, 3=Other, 0=nil |
| NNN | 3 | Sequential roll number (001–999) |

### NIM Format: `2209NNNN`

Fixed prefix `2209` + 4-digit sequential number zero-padded.

### Number Assignment Table

| # | Name | Role | Number | Meta | Del |
|---|------|------|--------|------|-----|
| 1 | Muhammad Arfan | super_admin | `200307212024073001` | `{"key":"A1B2C3D4"}` | — |
| 2 | Project | admin | `199205152013050002` | — | — |
| 3 | Edy Susilo | issuer | `198005172001051003` | `{"key":"E5F6G7H8"}` | — |
| 4 | Liesbeth Stifanny | holder | `22090001` | — | — |
| 5 | Anna Sorokin | holder | `22090002` | `{"key":"I9J0K1L2"}` | ✅ |
| 6–10 | Random | issuer (NIP) | NIP from DOB+recruit+gender+seq | 2 have meta | — |
| 11–15 | Random | holder (NIM) | `2209`0003–`2209`0007 | 2 have meta | ✅ users 7–10 |

Project's DOB/recruit: deterministic from `seedHashSeed("credchain-seed")` — DOB `1992-05-15`, recruit `2013-05` (21 years later), gender digit `0` (no gender set).

Meta defined users: `A1B2C3D4`, `E5F6G7H8`, `I9J0K1L2` (hardcoded, uppercase alphanumeric). Random users: generated via `seedRandomAlphaKey(rng)` producing 8-char uppercase alphanumeric `[A-Z0-9]{8}`.

Soft-delete targets: Anna Sorokin + 4 random users (last 4 in the random list: indices 6–9, wallet indices 12–15).

- [ ] **Step 1: Write updated tests FIRST**

```go
// infrastructure/database/seeder/user_seeder_test.go
package seeder_test

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/infrastructure/testutil/db"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestUserSeeder_Seeds15Users(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	ctx := context.Background()

	seedMnemonic := "test test test test test test test test test test test junk"
	encKey := string(fixtures.TestWalletEncryptionKey())
	s := seeder.NewUserSeeder(userRepo, seedMnemonic, encKey)

	err := s.Seed(ctx)
	assert.NoError(t, err)

	// Defined users exist with correct roles
	superAdmins, err := userRepo.FindByRole(ctx, domain.RoleSuperAdmin)
	assert.NoError(t, err)
	assert.Len(t, superAdmins, 1)
	assert.Equal(t, "arfan2173@gmail.com", superAdmins[0].Email)
	assert.Equal(t, "Muhammad Arfan", *superAdmins[0].Name)
	assert.NotNil(t, superAdmins[0].Number)
	assert.True(t, len(*superAdmins[0].Number) == 18) // NIP length
	assert.NotNil(t, superAdmins[0].Meta)
	assert.Equal(t, "A1B2C3D4", superAdmins[0].Meta["key"])
	assert.Nil(t, superAdmins[0].DeletedAt) // active

	admins, err := userRepo.FindByRole(ctx, domain.RoleAdmin)
	assert.NoError(t, err)
	assert.Len(t, admins, 1)
	assert.Equal(t, "arfanforproject@gmail.com", admins[0].Email)
	assert.NotNil(t, admins[0].Number)             // has NIP
	assert.Nil(t, admins[0].Meta)                   // no meta
	assert.Nil(t, admins[0].DeletedAt)              // active

	issuers, err := userRepo.FindByRole(ctx, domain.RoleIssuer)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(issuers), 1)
	hasEdy := false
	for _, u := range issuers {
		assert.NotNil(t, u.Number, "all users must have Number")
		assert.True(t, len(*u.Number) == 18, "issuer number must be 18-digit NIP")
		if u.Email == "edysusilo17580@gmail.com" {
			hasEdy = true
			assert.NotNil(t, u.Meta)
			assert.Equal(t, "E5F6G7H8", u.Meta["key"])
			assert.Nil(t, u.DeletedAt) // active
		}
	}
	assert.True(t, hasEdy, "Edy Susilo should be an issuer")

	holders, err := userRepo.FindByRole(ctx, domain.RoleHolder)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(holders), 2)
	for _, u := range holders {
		assert.NotNil(t, u.Number, "all users must have Number")
		// NIM format: 2209 + 4 digits
		assert.True(t, len(*u.Number) == 8)
		assert.True(t, (*u.Number)[:4] == "2209")
	}

	// Total is exactly 15
	total := len(superAdmins) + len(admins) + len(issuers) + len(holders)
	assert.Equal(t, 15, total)

	// 5 deleted, 10 active
	deletedCount := 0
	for _, groups := range [][]domain.User{superAdmins, admins, issuers, holders} {
		for _, u := range groups {
			if u.DeletedAt != nil {
				deletedCount++
			}
		}
	}
	assert.Equal(t, 5, deletedCount)

	// Anna should be deleted
	annaDeleted := false
	for _, u := range holders {
		if u.Email == "annasorokin2173@gmail.com" && u.DeletedAt != nil {
			annaDeleted = true
		}
	}
	assert.True(t, annaDeleted, "Anna Sorokin should be soft-deleted")
}

func TestUserSeeder_Name(t *testing.T) {
	s := seeder.NewUserSeeder(nil, "", "")
	assert.Equal(t, "user", s.Name())
}

func TestUserSeeder_DeterministicRandomUsers(t *testing.T) {
	gormDB1 := db.OpenInMemorySQLite(t)
	gormDB2 := db.OpenInMemorySQLite(t)

	repo1 := user.NewGormUserRepository(gormDB1)
	repo2 := user.NewGormUserRepository(gormDB2)

	encKey := string(fixtures.TestWalletEncryptionKey())
	mnemonic := "test test test test test test test test test test test junk"

	ctx := context.Background()

	s1 := seeder.NewUserSeeder(repo1, mnemonic, encKey)
	err := s1.Seed(ctx)
	assert.NoError(t, err)

	s2 := seeder.NewUserSeeder(repo2, mnemonic, encKey)
	err = s2.Seed(ctx)
	assert.NoError(t, err)

	issuers1, _ := repo1.FindByRole(ctx, domain.RoleIssuer)
	issuers2, _ := repo2.FindByRole(ctx, domain.RoleIssuer)
	holders1, _ := repo1.FindByRole(ctx, domain.RoleHolder)
	holders2, _ := repo2.FindByRole(ctx, domain.RoleHolder)

	assert.Equal(t, len(issuers1), len(issuers2))
	assert.Equal(t, len(holders1), len(holders2))

	for i := range issuers1 {
		assert.Equal(t, issuers1[i].Email, issuers2[i].Email)
		assert.Equal(t, issuers1[i].Number, issuers2[i].Number)
	}
	for i := range holders1 {
		assert.Equal(t, holders1[i].Email, holders2[i].Email)
		assert.Equal(t, holders1[i].Number, holders2[i].Number)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./infrastructure/database/seeder/ -run TestUserSeeder -v
```

Expected: FAIL — tests fail because the seeder doesn't yet produce numbers/meta/deleted users.

- [ ] **Step 3: Rewrite user_seeder.go**

```go
// infrastructure/database/seeder/user_seeder.go
package seeder

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"CredChain_Golang/domain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

type UserSeeder struct {
	repo       domain.UserRepository
	mnemonic   string
	encryptKey string
}

func NewUserSeeder(repo domain.UserRepository, mnemonic string, encryptKey string) *UserSeeder {
	return &UserSeeder{repo: repo, mnemonic: mnemonic, encryptKey: encryptKey}
}

func (s *UserSeeder) Name() string { return "user" }

func (s *UserSeeder) Seed(ctx context.Context) error {
	seed := seedHashSeed("credchain-seed")
	rng := rand.New(rand.NewSource(seed))
	gofakeit.Seed(seed)

	users := s.seedBuildUsers(rng)
	_, err := s.repo.Store(ctx, users...)
	if err != nil {
		return fmt.Errorf("user seeder: store: %w", err)
	}

	// Soft-delete 5 users: Anna (index 4) + 4 random (indices 10-13)
	deleteIDs := make([]string, 0, 5)
	deleteIDs = append(deleteIDs, users[4].Id) // Anna Sorokin
	for i := 10; i <= 13; i++ {
		deleteIDs = append(deleteIDs, users[i].Id)
	}

	if _, err := s.repo.Delete(ctx, deleteIDs...); err != nil {
		return fmt.Errorf("user seeder: delete: %w", err)
	}

	return nil
}

func (s *UserSeeder) seedBuildUsers(rng *rand.Rand) []domain.User {
	var nipSeq, nimSeq int
	users := make([]domain.User, 15)

	// --- Defined users (indices 0–4, wallets 1–5) ---

	users[0] = s.seedBuildUser(seedBuildUserParams{
		index: 1, name: "Muhammad Arfan", email: "arfan2173@gmail.com",
		phoneNumber: lo.ToPtr("+6289506089254"), birthDate: seedMustParseDate("2003-07-21"),
		gender:  seedGenderPtr(domain.GenderOther),
		meta:    map[string]any{"key": "A1B2C3D4"},
		role:    domain.RoleSuperAdmin,
		number:  seedGenerateNIP(time.Date(2003, 7, 21, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderOther), &nipSeq),
	})

	users[1] = s.seedBuildUser( seedBuildUserParams{
		index: 2, name: "Project", email: "arfanforproject@gmail.com",
		birthDate: seedMustParseDate("1992-05-15"),
		role:      domain.RoleAdmin,
		number:    seedGenerateNIP(time.Date(1992, 5, 15, 0, 0, 0, 0, time.UTC), nil, &nipSeq),
	})

	users[2] = s.seedBuildUser( seedBuildUserParams{
		index: 3, name: "Edy Susilo", email: "edysusilo17580@gmail.com",
		phoneNumber: lo.ToPtr("+6285228296172"), birthDate: seedMustParseDate("1980-05-17"),
		gender: seedGenderPtr(domain.GenderMale),
		meta:   map[string]any{"key": "E5F6G7H8"},
		role:   domain.RoleIssuer,
		number: seedGenerateNIP(time.Date(1980, 5, 17, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderMale), &nipSeq),
	})

	users[3] = s.seedBuildUser( seedBuildUserParams{
		index: 4, name: "Liesbeth Stifanny", email: "liesbethsh19@gmail.com",
		phoneNumber: lo.ToPtr("+6289676624902"), birthDate: seedMustParseDate("2003-09-19"),
		gender: seedGenderPtr(domain.GenderFemale),
		role:   domain.RoleHolder,
		number: seedGenerateNIM(&nimSeq),
	})

	users[4] = s.seedBuildUser( seedBuildUserParams{
		index: 5, name: "Anna Sorokin", email: "annasorokin2173@gmail.com",
		gender: seedGenderPtr(domain.GenderFemale),
		meta:   map[string]any{"key": "I9J0K1L2"},
		role:   domain.RoleHolder,
		number: seedGenerateNIM(&nimSeq),
	})

	// --- Random users (indices 5–14, wallets 6–15) ---
	for i := range 10 {
		idx := i + 5 // positions 5-14
		walletIdx := uint32(i + 6)
		role := seedRandomUserRole(rng)
		name := seedRandomIndonesianName(rng)
		email := seedNameToEmail(name)
		phone := SanitizePhone(seedRandomIndonesianPhone(rng))
		birthDate := seedRandomBirthDate(rng)
		gender := seedRandomGender(rng)

		var meta map[string]any
		// Alternating: even-index random users get meta (5 of 10)
		if i%2 == 0 {
			meta = map[string]any{"key": seedRandomAlphaKey(rng)}
		}

		var number string
		if role == domain.RoleIssuer {
			number = seedGenerateNIP(birthDate, &gender, &nipSeq)
		} else {
			number = seedGenerateNIM(&nimSeq)
		}

		users[idx] = s.seedBuildUser( seedBuildUserParams{
			index:       walletIdx,
			name:        name,
			email:       email,
			phoneNumber: &phone,
			birthDate:   &birthDate,
			gender:      &gender,
			meta:        meta,
			role:        role,
			number:      number,
		})
	}

	return users
}

type seedBuildUserParams struct {
	index       uint32
	name        string
	email       string
	phoneNumber *string
	birthDate   *time.Time
	gender      *domain.Gender
	meta        map[string]any
	role        domain.Role
	number      string
}

func (s *UserSeeder) seedBuildUser(p seedBuildUserParams) domain.User {
	privKeyHex, address, err := cryptoInfra.DeriveKeyFromMnemonic(s.mnemonic, p.index)
	if err != nil {
		panic(fmt.Sprintf("failed to derive key for user index %d: %v", p.index, err))
	}

	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKeyHex), []byte(s.encryptKey))
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt key for user index %d: %v", p.index, err))
	}

	return domain.User{
		Name:                      lo.ToPtr(p.name),
		Number:                    lo.ToPtr(p.number),
		PhoneNumber:               p.phoneNumber,
		Email:                     p.email,
		Gender:                    p.gender,
		BirthDate:                 p.birthDate,
		Meta:                      p.meta,
		Role:                      p.role,
		WalletAddress:             address,
		EncryptedWalletPrivateKey: encryptedKey,
	}
}

// --- NIP/NIM Generators ---

// seedGenerateNIP generates an 18-digit NIP: YYYYMMDD + YYYYMM(recruit=DOB+21y) + X(gender) + NNN(seq).
func seedGenerateNIP(dob time.Time, gender *domain.Gender, seq *int) string {
	recruit := dob.AddDate(21, 0, 0)
	genderDigit := '0'
	if gender != nil {
		switch *gender {
		case domain.GenderMale:
			genderDigit = '1'
		case domain.GenderFemale:
			genderDigit = '2'
		default:
			genderDigit = '3'
		}
	}
	*seq++
	return fmt.Sprintf("%s%04d%02d%c%03d",
		dob.Format("20060102"),
		recruit.Year(), recruit.Month(),
		genderDigit,
		*seq,
	)
}

// seedGenerateNIM generates an 8-digit NIM: 2209 + 4-digit zero-padded seq.
func seedGenerateNIM(seq *int) string {
	*seq++
	return fmt.Sprintf("2209%04d", *seq)
}

// seedRandomAlphaKey generates an 8-char uppercase alphanumeric string [A-Z0-9]{8}.
func seedRandomAlphaKey(rng *rand.Rand) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}

// --- Helpers (all prefixed with "seed") ---

func seedMustParseDate(date string) *time.Time {
	t, _ := time.Parse(time.DateOnly, date)
	return &t
}

func seedGenderPtr(g domain.Gender) *domain.Gender { return &g }

func seedHashSeed(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}

var seedIndoFirstNames = []string{
	"Ahmad", "Budi", "Citra", "Dewi", "Eko", "Fitri", "Gunawan", "Hadi", "Indah", "Joko",
	"Kartika", "Lestari", "Mega", "Nur", "Putri", "Rizky", "Sari", "Tono", "Wati", "Yanto",
}

var seedIndoLastNames = []string{
	"Santoso", "Wijaya", "Pratama", "Kusuma", "Hidayat", "Saputra", "Nugroho", "Permana",
	"Mahendra", "Setiawan", "Purnama", "Gunawan", "Hartono", "Wibowo", "Kurniawan",
}

func seedRandomIndonesianName(rng *rand.Rand) string {
	first := seedIndoFirstNames[rng.Intn(len(seedIndoFirstNames))]
	last := seedIndoLastNames[rng.Intn(len(seedIndoLastNames))]
	return first + " " + last
}

func seedNameToEmail(name string) string {
	parts := seedSplitName(name)
	email := ""
	for i, p := range parts {
		if i > 0 {
			email += "."
		}
		email += seedToLower(p)
	}
	return email + "@gmail.com"
}

func seedSplitName(name string) []string {
	var parts []string
	current := ""
	for _, r := range name {
		if r == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func seedToLower(s string) string {
	b := make([]byte, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			b[i] = byte(r + 32)
		} else {
			b[i] = byte(r)
		}
	}
	return string(b)
}

var seedIndoPhonePrefixes = []string{"812", "813", "821", "822", "823", "851", "852", "853", "856", "857", "858", "878", "895", "896", "897", "898", "899"}

func seedRandomIndonesianPhone(rng *rand.Rand) string {
	prefix := seedIndoPhonePrefixes[rng.Intn(len(seedIndoPhonePrefixes))]
	suffix := make([]byte, 8)
	for i := range suffix {
		suffix[i] = byte('0' + rng.Intn(10))
	}
	return "0" + prefix + string(suffix)
}

func seedRandomBirthDate(rng *rand.Rand) time.Time {
	min := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2005, 12, 31, 0, 0, 0, 0, time.UTC)
	delta := max.Unix() - min.Unix()
	sec := rng.Int63n(delta)
	return min.Add(time.Duration(sec) * time.Second)
}

var seedGenders = []domain.Gender{domain.GenderMale, domain.GenderFemale, domain.GenderOther}

func seedRandomGender(rng *rand.Rand) domain.Gender {
	return seedGenders[rng.Intn(len(seedGenders))]
}

var seedUserRoles = []domain.Role{
	domain.RoleHolder, domain.RoleHolder, domain.RoleHolder,
	domain.RoleIssuer, domain.RoleIssuer,
}

func seedRandomUserRole(rng *rand.Rand) domain.Role {
	return seedUserRoles[rng.Intn(len(seedUserRoles))]
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./infrastructure/database/seeder/ -run TestUserSeeder -v
```

Expected: PASS for all 3 tests.

- [ ] **Step 5: Commit**

```bash
git add infrastructure/database/seeder/user_seeder.go \
        infrastructure/database/seeder/user_seeder_test.go
git commit -m "feat: enhance seeder with NIP/NIM, meta, and soft-delete"
```

---

## Task 3: Seed-Chain — Process Deleted Users

**Files:**
- Modify: `cmd/seed_chain.go:92-99`

Change the filter logic so deleted users get `RoleNone` on-chain while preserving their DB role.

- [ ] **Step 1: Edit seed_chain.go filter logic**

Replace lines 93-99:

```go
	// Filter out users with RoleNone (no on-chain effect) and RoleSuperAdmin
	// (SuperAdmin is set during contract deployment and cannot be modified via
	// batchUpdateUserRoleWithSignature — contract reverts with
	// SuperAdminRoleNotUpdatableError).
	usersToRegister := lo.Filter(allUsers, func(u domain.User, _ int) bool {
		return u.Role != domain.RoleNone && u.Role != domain.RoleSuperAdmin
	})
```

With:

```go
	// Build the on-chain update list. SuperAdmin is skipped (set during deploy,
	// contract blocks SuperAdmin updates via batchUpdateUserRoleWithSignature).
	// Deleted users get RoleNone on-chain while preserving their DB role.
	// Active users get their DB role as-is.
	var usersToRegister []domain.User
	for _, u := range allUsers {
		if u.Role == domain.RoleSuperAdmin {
			continue
		}
		update := u
		if u.DeletedAt != nil {
			update.Role = domain.RoleNone
		}
		usersToRegister = append(usersToRegister, update)
	}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

Expected: builds without errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/seed_chain.go
git commit -m "feat: seed-chain sets RoleNone on-chain for deleted users"
```

---

## Task 4: Final Verification

**No files changed.** Run full test suite and static analysis.

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v 2>&1 | tail -30
```

Expected: all tests pass.

- [ ] **Step 2: Run go vet**

```bash
go vet ./...
```

Expected: zero output.

- [ ] **Step 3: Run gofmt check**

```bash
gofmt -l .
```

Expected: zero output (no unformatted files).

- [ ] **Step 4: Build binary**

```bash
go build -o bin/credchain main.go
```

Expected: builds successfully.

- [ ] **Step 5: Commit**

```bash
git add -A
git diff --cached --stat
git commit -m "chore: final verification after enhanced seeder"
```

---

## Rerun Flow (Fresh Everything)

After implementation, reset and verify:

```bash
# 1. Kill old Hardhat node
kill $(lsof -ti :8545) 2>/dev/null

# 2. Database reset (migrate down then up to apply gender reorder)
cd CredChain_Golang
make migrate-down && make migrate-up

# 3. Start fresh Hardhat node
cd ../CredChain_Solidity
npx hardhat node &
sleep 3

# 4. Deploy contracts + capture addresses
DEPLOY_OUT=$(npx hardhat run scripts/deploy.ts --network localhost 2>&1)
AUTH=$(echo "$DEPLOY_OUT" | grep "CredentialAuthority" | awk '{print $NF}')
REG=$(echo "$DEPLOY_OUT" | grep "CredentialRegistry" | awk '{print $NF}')

# 5. Update Go .env with fresh contract addresses
cd ../CredChain_Golang
sed -i '' "s/^AUTHORITY_CONTRACT=.*/AUTHORITY_CONTRACT=$AUTH/" .env
sed -i '' "s/^REGISTRY_CONTRACT=.*/REGISTRY_CONTRACT=$REG/" .env

# 6. Seed + register on-chain
make seed && make seed-chain

# 7. Verify database (15 users, numbers, meta, 5 deleted)
docker compose exec -T postgres psql -U root -d credchain -c "
  SELECT email, role, number,
         meta IS NOT NULL as has_meta,
         deleted_at IS NOT NULL as deleted
  FROM users ORDER BY created_at;"
```
