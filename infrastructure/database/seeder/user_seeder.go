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
	seed := hashToSeed("credchain-seed")
	rng := rand.New(rand.NewSource(seed))
	gofakeit.Seed(seed)

	users := s.seedBuildUsers(rng)
	_, err := s.repo.Store(ctx, users...)
	if err != nil {
		return fmt.Errorf("user seeder: store: %w", err)
	}

	deleteIDs := make([]string, 0, 10)
	deleteIDs = append(deleteIDs, users[4].Id)
	for i := 10; i <= 18; i++ {
		deleteIDs = append(deleteIDs, users[i].Id)
	}
	if _, err := s.repo.Delete(ctx, deleteIDs...); err != nil {
		return fmt.Errorf("user seeder: delete: %w", err)
	}

	return nil
}

func (s *UserSeeder) seedBuildUsers(rng *rand.Rand) []domain.User {
	var nipSeq, nimSeq int
	users := make([]domain.User, 150)

	users[0] = s.seedBuildUser(seedBuildUserParams{
		index: 1, name: "Muhammad Arfan", email: "arfan2173@gmail.com",
		phoneNumber: lo.ToPtr("+6289506089254"), birthDate: seedMustParseDate("2003-07-21"),
		gender: seedGenderPtr(domain.GenderOther),
		meta:   map[string]any{"key": "A1B2C3D4"},
		role:   domain.RoleSuperAdmin,
		number: seedGenerateNIP(time.Date(2003, 7, 21, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderOther), &nipSeq),
	})

	users[1] = s.seedBuildUser(seedBuildUserParams{
		index: 2, name: "Project", email: "arfanforproject@gmail.com",
		birthDate: seedMustParseDate("1992-05-15"),
		role:      domain.RoleAdmin,
		number:    seedGenerateNIP(time.Date(1992, 5, 15, 0, 0, 0, 0, time.UTC), nil, &nipSeq),
	})

	users[2] = s.seedBuildUser(seedBuildUserParams{
		index: 3, name: "Edy Susilo", email: "edysusilo17580@gmail.com",
		phoneNumber: lo.ToPtr("+6285228296172"), birthDate: seedMustParseDate("1980-05-17"),
		gender: seedGenderPtr(domain.GenderMale),
		meta:   map[string]any{"key": "E5F6G7H8"},
		role:   domain.RoleIssuer,
		number: seedGenerateNIP(time.Date(1980, 5, 17, 0, 0, 0, 0, time.UTC), seedGenderPtr(domain.GenderMale), &nipSeq),
	})

	users[3] = s.seedBuildUser(seedBuildUserParams{
		index: 4, name: "Liesbeth Stifanny", email: "liesbethsh19@gmail.com",
		phoneNumber: lo.ToPtr("+6289676624902"), birthDate: seedMustParseDate("2003-09-19"),
		gender: seedGenderPtr(domain.GenderFemale),
		role:   domain.RoleHolder,
		number: seedGenerateNIM(&nimSeq),
	})

	users[4] = s.seedBuildUser(seedBuildUserParams{
		index: 5, name: "Anna Sorokin", email: "annasorokin2173@gmail.com",
		gender: seedGenderPtr(domain.GenderFemale),
		meta:   map[string]any{"key": "I9J0K1L2"},
		role:   domain.RoleHolder,
		number: seedGenerateNIM(&nimSeq),
	})

	for i := range 145 {
		idx := i + 5
		walletIdx := uint32(i + 6)
		role := seedRandomUserRole(rng)
		name := seedRandomIndonesianName(rng)
		email := seedNameToEmail(name, idx)
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
		panic(fmt.Sprintf("failed to derive key for index %d: %v", p.index, err))
	}
	encryptedKey, err := cryptoInfra.Encrypt([]byte(privKeyHex), []byte(s.encryptKey))
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt key for index %d: %v", p.index, err))
	}
	return domain.User{
		Name: lo.ToPtr(p.name), Number: lo.ToPtr(p.number),
		PhoneNumber: p.phoneNumber, Email: p.email,
		Gender: p.gender, BirthDate: p.birthDate,
		Meta: p.meta, Role: p.role,
		WalletAddress: address, EncryptedWalletPrivateKey: encryptedKey,
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

func seedNameToEmail(name string, idx int) string {
	parts := seedSplitName(name)
	email := ""
	for i, p := range parts {
		if i > 0 {
			email += "."
		}
		email += seedToLower(p)
	}
	return fmt.Sprintf("%s.%d@gmail.com", email, idx)
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
	domain.RoleHolder, domain.RoleHolder, domain.RoleHolder, domain.RoleHolder,
	domain.RoleIssuer,
}

func seedRandomUserRole(rng *rand.Rand) domain.Role {
	return seedUserRoles[rng.Intn(len(seedUserRoles))]
}
