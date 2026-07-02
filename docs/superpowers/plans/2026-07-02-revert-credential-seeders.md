# Revert Credential Seeders & Scale Back User Seeder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove credential, extraction, and diploma seeders entirely. Scale UserSeeder back from 150 to 15 users with 5 soft-deletes. Add deterministic timestamps.

**Architecture:** Back out code introduced by commits `7feacbc`, `839316e`, `52d2474`, `fbc162f`, `a287c78`. Target state is commit `14d8561` (NIP/NIM/meta + 15 users + 5 soft-deletes) with `aceb5f9` rename preserved, plus new deterministic timestamp logic. Pre-set CreatedAt/UpdatedAt/DeletedAt on domain.User objects so GORM uses explicit values instead of auto-generated timestamps.

**Tech Stack:** Go 1.25, GORM, Uber FX, math/rand (deterministic), FNV-64a hash

## Global Constraints

- Go module path: `CredChain_Golang`
- Role enum: None(0) → Holder(1) → Issuer(2) → Admin(3) → SuperAdmin(4)
- All randomness must be deterministic: seeded via `hashToSeed("credchain-seed")` → `rand.New(rand.NewSource(seed))`
- No `time.Now()` in seeder code; all timestamps deterministically derived
- Keep: NIP/NIM, meta, phone numbers, chunked role registration (`maxBatchRole = 100`), anvil gas limit (`--gas-limit 100000000`), phone sanitizer
- Verification: `go test ./... && go vet ./... && gofmt -l .` (last must produce zero output)

---

### Task 1: Delete diploma rendering source files

**Files:**
- Delete: `infrastructure/database/seeder/diploma.go`
- Delete: `infrastructure/database/seeder/diploma_test.go`

**Interfaces:**
- Produces: None (these files are removed)
- Consumes: None

- [ ] **Step 1: Delete diploma source files**

```bash
rm infrastructure/database/seeder/diploma.go
```

- [ ] **Step 2: Delete diploma test file**

```bash
rm infrastructure/database/seeder/diploma_test.go
```

- [ ] **Step 3: Verify files removed**

```bash
ls infrastructure/database/seeder/diploma.go infrastructure/database/seeder/diploma_test.go
```

Expected: "No such file or directory" for both.

- [ ] **Step 4: Commit**

```bash
git add infrastructure/database/seeder/diploma.go infrastructure/database/seeder/diploma_test.go
git commit -m "revert(seeder): remove diploma rendering source files"
```

---

### Task 2: Delete credential seeder source files

**Files:**
- Delete: `infrastructure/database/seeder/credential_seeder.go`
- Delete: `infrastructure/database/seeder/credential_seeder_test.go`

**Interfaces:**
- Produces: None
- Consumes: None

- [ ] **Step 1: Delete credential seeder files**

```bash
rm infrastructure/database/seeder/credential_seeder.go
rm infrastructure/database/seeder/credential_seeder_test.go
```

- [ ] **Step 2: Commit**

```bash
git add infrastructure/database/seeder/credential_seeder.go infrastructure/database/seeder/credential_seeder_test.go
git commit -m "revert(seeder): remove credential seeder source files"
```

---

### Task 3: Delete credential extraction seeder source files

**Files:**
- Delete: `infrastructure/database/seeder/credential_extraction_seeder.go`
- Delete: `infrastructure/database/seeder/credential_extraction_seeder_test.go`

**Interfaces:**
- Produces: None
- Consumes: None

- [ ] **Step 1: Delete extraction seeder files**

```bash
rm infrastructure/database/seeder/credential_extraction_seeder.go
rm infrastructure/database/seeder/credential_extraction_seeder_test.go
```

- [ ] **Step 2: Commit**

```bash
git add infrastructure/database/seeder/credential_extraction_seeder.go infrastructure/database/seeder/credential_extraction_seeder_test.go
git commit -m "revert(seeder): remove credential extraction seeder source files"
```

---

### Task 4: Revert user_seeder.go — scale 150→15, 10→5 soft-deletes, remove gofakeit

**Files:**
- Modify: `infrastructure/database/seeder/user_seeder.go`

**Interfaces:**
- Produces: `UserSeeder` seeding 15 users with 5 soft-deletes, deterministic timestamps
- Consumes: `hashToSeed`, `SanitizePhone` (already defined in package)

The file is rewritten to: 15 users (5 defined + 10 random), 5 soft-deletes (Anna at index 4 + indices 10-13), email without idx suffix, 3 Holders + 2 Issuers in `seedUserRoles`. Remove `gofakeit` import. Remove uncommitted `FindByEmails` idempotency check. Add deterministic timestamps.

- [ ] **Step 1: Write the full replacement file**

Replace entire `infrastructure/database/seeder/user_seeder.go`:

```go
package seeder

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"CredChain_Golang/domain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

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
	seed := hashToSeed("credchain-seed")
	rng := rand.New(rand.NewSource(seed))

	users := s.seedBuildUsers(rng)

	_, err := s.repo.Store(ctx, users...)
	if err != nil {
		return fmt.Errorf("user seeder: store: %w", err)
	}

	return nil
}

func (s *UserSeeder) seedBuildUsers(rng *rand.Rand) []domain.User {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var nipSeq, nimSeq int
	users := make([]domain.User, 15)

	// 5 defined users (slots 0-4)

	createdAt0 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[0] = s.seedBuildUser(seedBuildUserParams{
		index: 1, name: "Muhammad Arfan", email: "arfan2173@gmail.com",
		phoneNumber: lo.ToPtr("+6289506089254"), birthDate: seedMustParseDate("2003-07-21"),
		gender: seedGenderPtr(domain.GenderOther),
		meta:   map[string]any{"key": "A1B2C3D4"},
		role:   domain.RoleSuperAdmin,
		number: seedGenerateNIP(time.Date(2003, 7, 21, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderOther), &nipSeq),
		createdAt: createdAt0,
		updatedAt: lo.ToPtr(createdAt0.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)),
	})

	createdAt1 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[1] = s.seedBuildUser(seedBuildUserParams{
		index: 2, name: "Project", email: "arfanforproject@gmail.com",
		birthDate: seedMustParseDate("1992-05-15"),
		role:      domain.RoleAdmin,
		number:    seedGenerateNIP(time.Date(1992, 5, 15, 0, 0, 0, 0, time.UTC), nil, &nipSeq),
		createdAt: createdAt1,
		updatedAt: lo.ToPtr(createdAt1.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)),
	})

	createdAt2 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[2] = s.seedBuildUser(seedBuildUserParams{
		index: 3, name: "Edy Susilo", email: "edysusilo17580@gmail.com",
		phoneNumber: lo.ToPtr("+6285228296172"), birthDate: seedMustParseDate("1980-05-17"),
		gender: seedGenderPtr(domain.GenderMale),
		meta:   map[string]any{"key": "E5F6G7H8"},
		role:   domain.RoleIssuer,
		number: seedGenerateNIP(time.Date(1980, 5, 17, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderMale), &nipSeq),
		createdAt: createdAt2,
		updatedAt: lo.ToPtr(createdAt2.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour)),
	})

	createdAt3 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[3] = s.seedBuildUser(seedBuildUserParams{
		index: 4, name: "Liesbeth Stifanny", email: "liesbethsh19@gmail.com",
		phoneNumber: lo.ToPtr("+6289676624902"), birthDate: seedMustParseDate("2003-09-19"),
		gender: seedGenderPtr(domain.GenderFemale),
		role:   domain.RoleHolder,
		number: seedGenerateNIM(&nimSeq),
		createdAt: createdAt3,
		updatedAt: &createdAt3,
	})

	createdAt4 := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
	users[4] = s.seedBuildUser(seedBuildUserParams{
		index: 5, name: "Anna Sorokin", email: "annasorokin2173@gmail.com",
		gender: seedGenderPtr(domain.GenderFemale),
		meta:   map[string]any{"key": "I9J0K1L2"},
		role:   domain.RoleHolder,
		number: seedGenerateNIM(&nimSeq),
		createdAt: createdAt4,
		updatedAt: &createdAt4,
		deletedAt: lo.ToPtr(createdAt4.Add(time.Duration(1+rng.Int63n(180)) * 24 * time.Hour)),
	})

	// 10 random users (slots 5-14)
	for i := range 10 {
		idx := i + 5
		walletIdx := uint32(i + 6)
		role := seedRandomUserRole(rng)
		name := seedRandomIndonesianName(rng)
		email := seedNameToEmail(name)
		phone := SanitizePhone(seedRandomIndonesianPhone(rng))
		birthDate := seedRandomBirthDate(rng)
		gender := seedRandomGender(rng)

		var meta map[string]any
		if i%2 == 0 {
			meta = map[string]any{"key": seedRandomAlphaKey(rng)}
		}

		var number string
		if role == domain.RoleIssuer {
			number = seedGenerateNIP(birthDate, &gender, &nipSeq)
		} else {
			number = seedGenerateNIM(&nimSeq)
		}

		createdAt := baseTime.Add(time.Duration(rng.Int63n(365)) * 24 * time.Hour)
		var updatedAt *time.Time
		if role == domain.RoleIssuer {
			updatedAt = lo.ToPtr(createdAt.Add(time.Duration(1+rng.Int63n(30)) * 24 * time.Hour))
		} else {
			updatedAt = &createdAt
		}

		var deletedAt *time.Time
		// users at indices 10-13 get soft-deleted
		if i >= 5 {
			t := updatedAt
			if t == nil {
				t = &createdAt
			}
			deletedAt = lo.ToPtr((*t).Add(time.Duration(1+rng.Int63n(180)) * 24 * time.Hour))
		}

		users[idx] = s.seedBuildUser(seedBuildUserParams{
			index:       walletIdx,
			name:        name,
			email:       email,
			phoneNumber: &phone,
			birthDate:   &birthDate,
			gender:      &gender,
			meta:        meta,
			role:        role,
			number:      number,
			createdAt:   createdAt,
			updatedAt:   updatedAt,
			deletedAt:   deletedAt,
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
	createdAt   time.Time
	updatedAt   *time.Time
	deletedAt   *time.Time
}

func (s *UserSeeder) seedBuildUser(p seedBuildUserParams) domain.User {
	privKeyHex, address, err := cryptoInfra.DeriveKeyFromMnemonic(s.mnemonic, p.index)
	if err != nil {
		panic(fmt.Sprintf("failed to derive key for index %d: %v", p.index, err))
	}
	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKeyHex), []byte(s.encryptKey))
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt key for index %d: %v", p.index, err))
	}
	updatedAt := p.updatedAt
	if updatedAt == nil {
		updatedAt = &p.createdAt
	}
	return domain.User{
		Name: lo.ToPtr(p.name), Number: lo.ToPtr(p.number),
		PhoneNumber: p.phoneNumber, Email: p.email,
		Gender: p.gender, BirthDate: p.birthDate,
		Meta: p.meta, Role: p.role,
		WalletAddress: address, EncryptedWalletPrivateKey: encryptedKey,
		CreatedAt: p.createdAt, UpdatedAt: updatedAt, DeletedAt: p.deletedAt,
	}
}

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
		dob.Format("20060102"), recruit.Year(), recruit.Month(), genderDigit, *seq)
}

func seedGenerateNIM(seq *int) string {
	*seq++
	return fmt.Sprintf("2209%04d", *seq)
}

func seedRandomAlphaKey(rng *rand.Rand) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}

func seedMustParseDate(date string) *time.Time {
	t, _ := time.Parse(time.DateOnly, date)
	return &t
}

func seedGenderPtr(g domain.Gender) *domain.Gender { return &g }

func hashToSeed(s string) int64 {
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
	return seedIndoFirstNames[rng.Intn(len(seedIndoFirstNames))] + " " + seedIndoLastNames[rng.Intn(len(seedIndoLastNames))]
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
	return min.Add(time.Duration(rng.Int63n(delta)) * time.Second)
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

Key changes from current code:
- Lines 30-35: Removed `gofakeit.Seed(seed)` call
- Lines 36-61: Removed entire `FindByEmails` idempotency check + `newUsers` loop → single `Store` call
- Lines 63-70: Removed `deleteIDs`/`Delete` block → soft-delete now via pre-set `DeletedAt` in user objects
- Line 77: `users := make([]domain.User, 150)` → `users := make([]domain.User, 15)`
- Lines 120-153: `for i := range 145` → `for i := range 10`
- Line 125: `seedNameToEmail(name, idx)` → `seedNameToEmail(name)`
- Lines 109-118: Timestamps computed deterministically via `rng.Int63n(365)` offset from `baseTime`
- Lines 111-118: Soft-deleted users (indices 10-13) pre-set `DeletedAt` > `UpdatedAt`
- Line 313-316: `seedUserRoles` → 3 Holders + 2 Issuers (60/40 tilt)
- `seedBuildUserParams`: added `createdAt`, `updatedAt`, `deletedAt` fields
- `seedBuildUser`: returns `domain.User` with `CreatedAt`, `UpdatedAt`, `DeletedAt`
- `seedNameToEmail`: signature changed from `(name string, idx int)` to `(name string)` — no more idx suffix

- [ ] **Step 2: Verify imports are valid (no gofakeit)**

Check that `gofakeit` is not imported:

```bash
grep -n "gofakeit" infrastructure/database/seeder/user_seeder.go
```

Expected: No matches.

- [ ] **Step 3: Run tests**

```bash
go test ./infrastructure/database/seeder/... -v
```

Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add infrastructure/database/seeder/user_seeder.go
git commit -m "revert(seeder): scale user seeder 150→15, 10→5 soft-deletes, deterministic timestamps"
```

---

### Task 5: Update user_seeder_test.go — fix assertion values

**Files:**
- Modify: `infrastructure/database/seeder/user_seeder_test.go`

**Interfaces:**
- Produces: Updated test assertions for 15 users, 5 soft-deletes
- Consumes: `seeder.NewUserSeeder`

Three changes: test name 150→15, total assertion 150→15, deleted count 10→5.

- [ ] **Step 1: Rename test function**

Edit `infrastructure/database/seeder/user_seeder_test.go` — replace at line 16:

Old:
```go
func TestUserSeeder_Seeds150Users(t *testing.T) {
```

New:
```go
func TestUserSeeder_Seeds15Users(t *testing.T) {
```

- [ ] **Step 2: Fix total assertion**

Edit line 73 — replace:

Old:
```go
	assert.Equal(t, 150, total)
```

New:
```go
	assert.Equal(t, 15, total)
```

- [ ] **Step 3: Fix deleted count assertion**

Edit line 83 — replace:

Old:
```go
	assert.Equal(t, 10, deletedCount)
```

New:
```go
	assert.Equal(t, 5, deletedCount)
```

- [ ] **Step 4: Run tests to verify**

```bash
go test ./infrastructure/database/seeder/... -v
```

Expected: All tests PASS, including `TestUserSeeder_Seeds15Users` and `TestUserSeeder_DeterministicRandomUsers`.

- [ ] **Step 5: Commit**

```bash
git add infrastructure/database/seeder/user_seeder_test.go
git commit -m "test(seeder): update assertions for 15 users and 5 soft-deletes"
```

---

### Task 6: Clean cmd/seed.go — remove credential seeder registrations

**Files:**
- Modify: `cmd/seed.go`

**Interfaces:**
- Produces: `seedCmd` FX app with only UserSeeder in registry
- Consumes: `seeder.NewRegistry`, `seeder.NewUserSeeder`

Remove `NewCredentialSeeder`, `NewCredentialExtractionSeeder` registrations. Remove orphaned imports: `feature/credential`, `infraMongo`, `storage`. Remove orphaned FX providers: `credential.NewGormCredentialRepository`, `credential.NewMongoCredentialExtractionRepository`, `storage.NewStorage`, `infraMongo.NewClient`, `infraMongo.NewDatabase`.

- [ ] **Step 1: Replace imports block**

Replace lines 3-21:

Old:
```go
import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	infraMongo "CredChain_Golang/infrastructure/database/mongo"
	"CredChain_Golang/infrastructure/database/seeder"
	infraLogger "CredChain_Golang/infrastructure/logger"
	"CredChain_Golang/infrastructure/storage"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)
```

New:
```go
import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/seeder"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)
```

- [ ] **Step 2: Replace FX providers block**

Replace lines 50-66:

Old:
```go
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				infraMongo.NewClient,
				infraMongo.NewDatabase,
				user.NewGormUserRepository,
				credential.NewGormCredentialRepository,
				credential.NewMongoCredentialExtractionRepository,
				storage.NewStorage,
				func(userRepo domain.UserRepository, credentialRepo domain.CredentialRepository, extractionRepo domain.CredentialExtractionRepository, fs *storage.Storage, cfg *config.Config) *seeder.Registry {
					mnemonic := seedGetHardhatMnemonic(cfg)
					return seeder.NewRegistry(
						seeder.NewUserSeeder(userRepo, mnemonic, *cfg.WalletEncryptionKey),
						seeder.NewCredentialSeeder(credentialRepo, userRepo, fs, cfg, 1),
						seeder.NewCredentialExtractionSeeder(extractionRepo, credentialRepo),
					)
				},
			),
```

New:
```go
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
				func(userRepo domain.UserRepository, cfg *config.Config) *seeder.Registry {
					mnemonic := seedGetHardhatMnemonic(cfg)
					return seeder.NewRegistry(
						seeder.NewUserSeeder(userRepo, mnemonic, *cfg.WalletEncryptionKey),
					)
				},
			),
```

- [ ] **Step 3: Verify file compiles**

```bash
go build ./cmd/...
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add cmd/seed.go
git commit -m "revert(seed): remove credential and extraction seeder registrations from CLI"
```

---

### Task 7: Clean cmd/seed_chain.go — remove credential minting block

**Files:**
- Modify: `cmd/seed_chain.go`

**Interfaces:**
- Produces: `seedChainCmd` FX app without credential minting
- Consumes: `seedGetHardhatMnemonic` (from `cmd/seed.go`)

Remove credential minting block (lines 168-237). Remove orphaned imports: `domainQuery`, `feature/credential`, `chain.RegistryService`. Remove orphaned FX providers: `credential.NewGormCredentialRepository`, `chain.NewRegistryService`. Update function signature and logs.

- [ ] **Step 1: Replace imports**

Replace lines 3-21:

Old:
```go
import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)
```

New:
```go
import (
	"context"
	"fmt"
	"log"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/chain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)
```

- [ ] **Step 2: Replace FX providers block**

Replace lines 49-59:

Old:
```go
		app := fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
				credential.NewGormCredentialRepository,
				chain.NewClient,
				chain.NewAuthorityService,
				chain.NewRegistryService,
			),
```

New:
```go
		app := fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				gormInfra.NewGorm,
				user.NewGormUserRepository,
				chain.NewClient,
				chain.NewAuthorityService,
			),
```

- [ ] **Step 3: Replace Invoke function signature**

Replace lines 60-68:

Old:
```go
			fx.Invoke(func(shutdowner fx.Shutdowner, cfg *config.Config, userRepo domain.UserRepository, credentialRepo domain.CredentialRepository, authorityService chain.AuthorityService, registryService chain.RegistryService, logger *zap.Logger) {
				go func() {
					if err := seedChainRun(cfg, userRepo, credentialRepo, authorityService, registryService, seedChainNames, logger); err != nil {
						logger.Error("seed-chain failed", zap.Error(err))
					}
					shutdowner.Shutdown()
				}()
			}),
```

New:
```go
			fx.Invoke(func(shutdowner fx.Shutdowner, cfg *config.Config, userRepo domain.UserRepository, authorityService chain.AuthorityService, logger *zap.Logger) {
				go func() {
					if err := seedChainRun(cfg, userRepo, authorityService, seedChainNames, logger); err != nil {
						logger.Error("seed-chain failed", zap.Error(err))
					}
					shutdowner.Shutdown()
				}()
			}),
```

- [ ] **Step 4: Replace seedChainRun function signature and remove credential minting block**

Replace lines 78-245:

Old:
```go
// seedChainRun reads all users from PostgreSQL via a single Get query,
// derives the SuperAdmin wallet (mnemonic index 1), and registers every
// non-None-role user on-chain in one batch transaction.
//
// After role registration, it also mints credentials on-chain for any
// credentials whose TokenID is nil (not yet minted).
func seedChainRun(cfg *config.Config, userRepo domain.UserRepository, credentialRepo domain.CredentialRepository, authorityService chain.AuthorityService, registryService chain.RegistryService, names []string, logger *zap.Logger) error {
	ctx := context.Background()
	mnemonic := seedGetHardhatMnemonic(cfg)

	logger.Info("reading seeded users from database")

	allUsers, total, err := userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed-chain: read users: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("seed-chain: no users in database — run 'seed' first")
	}

	var usersToRegister []domain.User
	for _, u := range allUsers {
		if u.Role == domain.RoleSuperAdmin {
			continue
		}
		update := u
		if u.DeletedAt != nil {
			update.Role = domain.RoleNone
		}
		if update.Role == domain.RoleNone {
			continue
		}
		usersToRegister = append(usersToRegister, update)
	}

	logger.Info("users loaded for chain registration",
		zap.Int("total_in_db", total),
		zap.Int("to_register", len(usersToRegister)),
	)

	privKey, _, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, 1)
	if err != nil {
		return fmt.Errorf("seed-chain: derive super admin key: %w", err)
	}

	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKey), []byte(*cfg.WalletEncryptionKey))
	if err != nil {
		return fmt.Errorf("seed-chain: encrypt super admin key: %w", err)
	}

	addr, err := cryptoInfra.DeriveAddressFromPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("seed-chain: derive super admin address: %w", err)
	}

	superAdminWallet := domain.Wallet{
		Address:             addr,
		EncryptedPrivateKey: encryptedKey,
	}

	const maxBatchRole = 100
	registeredCount := 0
	for start := 0; start < len(usersToRegister); start += maxBatchRole {
		end := start + maxBatchRole
		if end > len(usersToRegister) {
			end = len(usersToRegister)
		}
		chunk := usersToRegister[start:end]

		logger.Info("registering users on-chain",
			zap.Int("chunk_size", len(chunk)),
			zap.Int("chunk_start", start),
			zap.Int("total", len(usersToRegister)),
			zap.String("signer", superAdminWallet.Address),
		)

		if err := authorityService.UpdateUserRole(ctx, superAdminWallet, chunk...); err != nil {
			return fmt.Errorf("seed-chain: on-chain registration chunk [%d:%d]: %w", start, end, err)
		}
		registeredCount += len(chunk)
	}

	// ── Credential minting ──────────────────────────────────────────────
	... all credential minting code ...

	logger.Info("seed-chain completed successfully",
		zap.Int("users_registered", registeredCount),
		zap.Int("credentials_minted", mintedCount),
	)

	return nil
}
```

New:
```go
// seedChainRun reads all users from PostgreSQL via a single Get query,
// derives the SuperAdmin wallet (mnemonic index 1), and registers every
// non-None-role user on-chain in chunked batches of ≤100.
func seedChainRun(cfg *config.Config, userRepo domain.UserRepository, authorityService chain.AuthorityService, names []string, logger *zap.Logger) error {
	ctx := context.Background()
	mnemonic := seedGetHardhatMnemonic(cfg)

	logger.Info("reading seeded users from database")

	allUsers, total, err := userRepo.Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed-chain: read users: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("seed-chain: no users in database — run 'seed' first")
	}

	var usersToRegister []domain.User
	for _, u := range allUsers {
		if u.Role == domain.RoleSuperAdmin {
			continue
		}
		update := u
		if u.DeletedAt != nil {
			update.Role = domain.RoleNone
		}
		if update.Role == domain.RoleNone {
			continue
		}
		usersToRegister = append(usersToRegister, update)
	}

	logger.Info("users loaded for chain registration",
		zap.Int("total_in_db", total),
		zap.Int("to_register", len(usersToRegister)),
	)

	privKey, _, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, 1)
	if err != nil {
		return fmt.Errorf("seed-chain: derive super admin key: %w", err)
	}

	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKey), []byte(*cfg.WalletEncryptionKey))
	if err != nil {
		return fmt.Errorf("seed-chain: encrypt super admin key: %w", err)
	}

	addr, err := cryptoInfra.DeriveAddressFromPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("seed-chain: derive super admin address: %w", err)
	}

	superAdminWallet := domain.Wallet{
		Address:             addr,
		EncryptedPrivateKey: encryptedKey,
	}

	const maxBatchRole = 100
	registeredCount := 0
	for start := 0; start < len(usersToRegister); start += maxBatchRole {
		end := start + maxBatchRole
		if end > len(usersToRegister) {
			end = len(usersToRegister)
		}
		chunk := usersToRegister[start:end]

		logger.Info("registering users on-chain",
			zap.Int("chunk_size", len(chunk)),
			zap.Int("chunk_start", start),
			zap.Int("total", len(usersToRegister)),
			zap.String("signer", superAdminWallet.Address),
		)

		if err := authorityService.UpdateUserRole(ctx, superAdminWallet, chunk...); err != nil {
			return fmt.Errorf("seed-chain: on-chain registration chunk [%d:%d]: %w", start, end, err)
		}
		registeredCount += len(chunk)
	}

	logger.Info("seed-chain completed successfully",
		zap.Int("users_registered", registeredCount),
	)

	return nil
}
```

- [ ] **Step 5: Verify file compiles**

```bash
go build ./cmd/...
```

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add cmd/seed_chain.go
git commit -m "revert(seed-chain): remove credential minting block"
```

---

### Task 8: Clean go.mod dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Produces: Cleaned go.mod without `go-pdf/fpdf` or `golang.org/x/image`
- Consumes: None

- [ ] **Step 1: Run go mod tidy**

```bash
go mod tidy
```

This automatically removes orphaned dependencies `github.com/go-pdf/fpdf` and `golang.org/x/image` from `go.mod` and `go.sum`.

- [ ] **Step 2: Verify only expected dependencies removed**

```bash
grep -c "go-pdf/fpdf" go.mod
```

Expected: 0.

```bash
grep -c "golang.org/x/image" go.mod
```

Expected: 0.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): remove orphaned diploma rendering dependencies"
```

---

### Task 9: Update Makefile help text

**Files:**
- Modify: `Makefile`

**Interfaces:**
- Produces: Accurate help text
- Consumes: None

- [ ] **Step 1: Update help text**

Edit `Makefile` lines 28-29 — replace:

Old:
```
	@echo "  make seed              - Run database seeders (populate users)"
	@echo "  make seed-chain        - Register seeded users on-chain"
```

New:
```
	@echo "  make seed              - Run database seeders (populate 15 users)"
	@echo "  make seed-chain        - Register seeded user roles on-chain"
```

- [ ] **Step 2: Commit**

```bash
git add Makefile
git commit -m "chore(makefile): update seed help text for 15 users"
```

---

### Task 10: Update AGENTS.md seeder description

**Files:**
- Modify: `AGENTS.md`

**Interfaces:**
- Produces: Accurate AGENTS.md
- Consumes: None

Three sections need updating: line 29 (command description), line 138 (directory listing), and line 434 (seeder documentation block).

- [ ] **Step 1: Update line 29 — make seed description**

Edit lines 29-30 — replace:

Old:
```
29: make seed                           # Run database seeders (populate 150 users)
30: make seed-chain                     # Register seeded users on-chain
```

New:
```
29: make seed                           # Run database seeders (populate 15 users)
30: make seed-chain                     # Register seeded user roles on-chain
```

- [ ] **Step 2: Update line 138 — directory listing**

Edit line 138 — replace:

Old:
```
138:     database/seeder/    → Seeder interface, Registry runner, UserSeeder, phone sanitizer
```

New:
```
138:     database/seeder/    → Seeder interface, Registry runner, UserSeeder (15 users), phone sanitizer
```

- [ ] **Step 3: Update line 434 — seeder documentation block**

Edit line 434 — replace:

Old (entire block at line 434):
```
**Database Seeder:** `infrastructure/database/seeder/` implements a `Seeder` interface with a `Registry` runner accepting variadic `--names` flags, executable via `make seed` and `make seed-chain`. The `UserSeeder` creates 150 users (5 defined + 145 randomised Indonesian names, 80/20 Holder/Issuer tilt) with wallet keys derived from the standard Hardhat mnemonic via BIP44 (`DeriveKeyFromMnemonic`). All users receive an employee number (NIP, 18-digit `YYYYMMDDYYYYMMXNNN`) for Issuer+ roles or a student number (NIM, `2209XXXX`) for Holder roles. Half the users receive random `{"key":"...}` metadata. Ten users are soft-deleted (Anna Sorokin at index 4 + 9 random users). Chain roles are registered via `make seed-chain`, which reads the database with a nil query and signs batch `UpdateUserRole` transactions in chunks of ≤100 (respecting `MAX_BATCH_ROLE=100` limit in the CredentialAuthority contract) with the SuperAdmin wallet (Hardhat node #1). SuperAdmin and users whose target role is `RoleNone` on a fresh deploy are skipped to avoid contract reverts (`SuperAdminRoleNotUpdatableError`, `SameRoleUpdateError`). The phone sanitizer (`SanitizePhone`) ensures E.164 compliance for all generated phone numbers.
```

New:
```
**Database Seeder:** `infrastructure/database/seeder/` implements a `Seeder` interface with a `Registry` runner accepting variadic `--names` flags, executable via `make seed` and `make seed-chain`. The `UserSeeder` creates 15 users (5 defined + 10 randomised Indonesian names, 60/40 Holder/Issuer tilt) with wallet keys derived from the standard Hardhat mnemonic via BIP44 (`DeriveKeyFromMnemonic`). All users receive an employee number (NIP, 18-digit `YYYYMMDDYYYYMMXNNN`) for Issuer+ roles or a student number (NIM, `2209XXXX`) for Holder roles. Half the users receive random `{"key":"...}` metadata. Five users are soft-deleted (Anna Sorokin at index 4 + 4 users at indices 10-13). All timestamps (created_at, updated_at, deleted_at) are deterministically generated from a seeded RNG. Chain roles are registered via `make seed-chain`, which reads the database with a nil query and signs batch `UpdateUserRole` transactions in chunks of ≤100 (respecting `MAX_BATCH_ROLE=100` limit in the CredentialAuthority contract) with the SuperAdmin wallet (Hardhat node #1). SuperAdmin and users whose target role is `RoleNone` on a fresh deploy are skipped to avoid contract reverts (`SuperAdminRoleNotUpdatableError`, `SameRoleUpdateError`). The phone sanitizer (`SanitizePhone`) ensures E.164 compliance for all generated phone numbers. Soft-deleted users are created with `DeletedAt` pre-set; `make seed-chain` detects these via `DeletedAt != nil` and assigns `RoleNone` on-chain.
```

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md
git commit -m "docs(agents): update seeder description for 15 users and 5 soft-deletes"
```

---

### Task 11: Delete credential seeding documentation files

**Files:**
- Delete: `docs/superpowers/specs/2026-07-01-credential-seeding-design.md`
- Delete: `docs/superpowers/plans/2026-07-01-credential-seeding.md`
- Delete: `docs/superpowers/plans/task-briefs/task-1-brief.md`
- Delete: `docs/superpowers/plans/task-briefs/task-1-report.md`
- Delete: `docs/superpowers/plans/task-briefs/task-2-brief.md`
- Delete: `docs/superpowers/plans/task-briefs/task-2-report.md`
- Delete: `docs/superpowers/plans/task-briefs/task-3-brief.md`
- Delete: `docs/superpowers/plans/task-briefs/task-3-report.md`

**Interfaces:**
- Produces: None
- Consumes: None

- [ ] **Step 1: Delete all credential seeding documentation**

```bash
rm docs/superpowers/specs/2026-07-01-credential-seeding-design.md
rm docs/superpowers/plans/2026-07-01-credential-seeding.md
rm docs/superpowers/plans/task-briefs/task-1-brief.md
rm docs/superpowers/plans/task-briefs/task-1-report.md
rm docs/superpowers/plans/task-briefs/task-2-brief.md
rm docs/superpowers/plans/task-briefs/task-2-report.md
rm docs/superpowers/plans/task-briefs/task-3-brief.md
rm docs/superpowers/plans/task-briefs/task-3-report.md
```

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/specs/2026-07-01-credential-seeding-design.md \
        docs/superpowers/plans/2026-07-01-credential-seeding.md \
        docs/superpowers/plans/task-briefs/task-1-brief.md \
        docs/superpowers/plans/task-briefs/task-1-report.md \
        docs/superpowers/plans/task-briefs/task-2-brief.md \
        docs/superpowers/plans/task-briefs/task-2-report.md \
        docs/superpowers/plans/task-briefs/task-3-brief.md \
        docs/superpowers/plans/task-briefs/task-3-report.md
git commit -m "docs(seeder): remove credential seeding documentation"
```

---

### Task 12: Full verification

**Files:**
- All modified files

**Interfaces:**
- Produces: Clean build and passing tests
- Consumes: All previous task outputs

- [ ] **Step 1: Run all tests**

```bash
go test ./... -count=1
```

Expected: All tests PASS.

- [ ] **Step 2: Run vet**

```bash
go vet ./...
```

Expected: Zero output (no warnings).

- [ ] **Step 3: Check formatting**

```bash
gofmt -l .
```

Expected: Zero output (all files formatted).

- [ ] **Step 4: Full build**

```bash
go build ./...
```

Expected: No errors.

- [ ] **Step 5: Verify git status is clean (no uncommitted changes beyond plan document)**

```bash
git status
```
