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
	users := s.seedBuildUsers()
	_, err := s.repo.Store(ctx, users...)
	if err != nil {
		return fmt.Errorf("user seeder: store: %w", err)
	}
	return nil
}

func (s *UserSeeder) seedBuildUsers() []domain.User {
	defined := s.seedBuildDefinedUsers()
	random := s.seedBuildRandomUsers()
	return append(defined, random...)
}

func (s *UserSeeder) seedBuildDefinedUsers() []domain.User {
	type definedUser struct {
		index       uint32
		name        *string
		email       string
		phoneNumber *string
		birthDate   *time.Time
		gender      *domain.Gender
		role        domain.Role
	}

	defs := []definedUser{
		{
			index: 1, name: lo.ToPtr("Muhammad Arfan"), email: "arfan2173@gmail.com",
			phoneNumber: lo.ToPtr("+6289506089254"), birthDate: seedMustParseDate("2003-07-21"),
			gender: seedGenderPtr(domain.GenderOther), role: domain.RoleSuperAdmin,
		},
		{
			index: 2, name: lo.ToPtr("Project"), email: "arfanforproject@gmail.com",
			role: domain.RoleAdmin,
		},
		{
			index: 3, name: lo.ToPtr("Edy Susilo"), email: "edysusilo17580@gmail.com",
			phoneNumber: lo.ToPtr("+6285228296172"), birthDate: seedMustParseDate("1980-05-17"),
			gender: seedGenderPtr(domain.GenderMale), role: domain.RoleIssuer,
		},
		{
			index: 4, name: lo.ToPtr("Liesbeth Stifanny"), email: "liesbethsh19@gmail.com",
			phoneNumber: lo.ToPtr("+6289676624902"), birthDate: seedMustParseDate("2003-09-19"),
			gender: seedGenderPtr(domain.GenderFemale), role: domain.RoleHolder,
		},
		{
			index: 5, name: lo.ToPtr("Anna Sorokin"), email: "annasorokin2173@gmail.com",
			gender: seedGenderPtr(domain.GenderFemale), role: domain.RoleHolder,
		},
	}

	users := make([]domain.User, len(defs))
	for i, d := range defs {
		privKeyHex, address, err := cryptoInfra.DeriveKeyFromMnemonic(s.mnemonic, d.index)
		if err != nil {
			panic(fmt.Sprintf("failed to derive key for defined user index %d: %v", d.index, err))
		}

		encryptedKey, err := cryptoInfra.Encrypt([]byte(privKeyHex), []byte(s.encryptKey))
		if err != nil {
			panic(fmt.Sprintf("failed to encrypt key for defined user index %d: %v", d.index, err))
		}

		phone := d.phoneNumber
		if phone != nil {
			phone = lo.ToPtr(SanitizePhone(*phone))
		}

		users[i] = domain.User{
			Name:                      d.name,
			Email:                     d.email,
			PhoneNumber:               phone,
			BirthDate:                 d.birthDate,
			Gender:                    d.gender,
			Role:                      d.role,
			WalletAddress:             address,
			EncryptedWalletPrivateKey: encryptedKey,
		}
	}

	return users
}

func (s *UserSeeder) seedBuildRandomUsers() []domain.User {
	seed := seedHashSeed("credchain-seed")
	rng := rand.New(rand.NewSource(seed))
	gofakeit.Seed(seed)

	users := make([]domain.User, 10)
	for i := range 10 {
		walletIdx := uint32(i + 6)
		privKeyHex, address, err := cryptoInfra.DeriveKeyFromMnemonic(s.mnemonic, walletIdx)
		if err != nil {
			panic(fmt.Sprintf("failed to derive key for random user index %d: %v", walletIdx, err))
		}

		encryptedKey, err := cryptoInfra.Encrypt([]byte(privKeyHex), []byte(s.encryptKey))
		if err != nil {
			panic(fmt.Sprintf("failed to encrypt key for random user index %d: %v", walletIdx, err))
		}

		name := seedRandomIndonesianName(rng)
		email := seedNameToEmail(name)
		phone := seedRandomIndonesianPhone(rng)
		phone = SanitizePhone(phone)
		birthDate := seedRandomBirthDate(rng)
		gender := seedRandomGender(rng)
		role := seedRandomUserRole(rng)

		users[i] = domain.User{
			Name:                      lo.ToPtr(name),
			Email:                     email,
			PhoneNumber:               lo.ToPtr(phone),
			BirthDate:                 &birthDate,
			Gender:                    &gender,
			Role:                      role,
			WalletAddress:             address,
			EncryptedWalletPrivateKey: encryptedKey,
		}
	}

	return users
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
