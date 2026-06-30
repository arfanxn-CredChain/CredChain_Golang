package seeder_test

import (
	"math/rand"
	"testing"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/seeder"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedBuildDiplomaContent(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	birthDate := time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC)

	holder := domain.User{
		Id:        "user-holder-001",
		Name:      strPtr("Budi Santoso"),
		BirthDate: &birthDate,
	}
	issuer := domain.User{
		Id:     "user-issuer-001",
		Name:   strPtr("Dr. Dewi Lestari"),
		Number: strPtr("196805152008123456"),
	}

	content := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rng)

	text := seeder.SeedBuildDiplomaText(content)
	assert.Contains(t, text, "Budi Santoso")
	assert.Contains(t, text, "Dr. Dewi Lestari")
	assert.Contains(t, text, "Teknik Informatika")
}

func TestSeedBuildDiplomaContent_NilNames(t *testing.T) {
	rng := rand.New(rand.NewSource(99))

	holder := domain.User{
		Id:   "user-holder-nil",
		Name: nil,
	}
	issuer := domain.User{
		Id:     "user-issuer-nil",
		Name:   nil,
		Number: nil,
	}

	content := seeder.SeedBuildDiplomaContent(holder, issuer, 5, rng)

	text := seeder.SeedBuildDiplomaText(content)
	assert.Contains(t, text, "—")
}

func TestSeedRenderDiplomaPNG(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	birthDate := time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC)

	holder := domain.User{
		Id:        "user-png-001",
		Name:      strPtr("Budi Santoso"),
		BirthDate: &birthDate,
	}
	issuer := domain.User{
		Id:     "user-png-002",
		Name:   strPtr("Dr. Dewi Lestari"),
		Number: strPtr("196805152008123456"),
	}

	content := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rng)
	pngBytes, err := seeder.SeedRenderDiplomaPNG(content)
	require.NoError(t, err)
	require.NotEmpty(t, pngBytes)

	assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, pngBytes[:4], "PNG magic number")
}

func TestSeedRenderDiplomaJPEG(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	birthDate := time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC)

	holder := domain.User{
		Id:        "user-jpg-001",
		Name:      strPtr("Budi Santoso"),
		BirthDate: &birthDate,
	}
	issuer := domain.User{
		Id:     "user-jpg-002",
		Name:   strPtr("Dr. Dewi Lestari"),
		Number: strPtr("196805152008123456"),
	}

	content := seeder.SeedBuildDiplomaContent(holder, issuer, 2, rng)
	jpgBytes, err := seeder.SeedRenderDiplomaJPEG(content)
	require.NoError(t, err)
	require.NotEmpty(t, jpgBytes)

	assert.Equal(t, []byte{0xFF, 0xD8, 0xFF}, jpgBytes[:3], "JPEG magic bytes")
}

func TestSeedRenderDiplomaPDF(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	birthDate := time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC)

	holder := domain.User{
		Id:        "user-pdf-001",
		Name:      strPtr("Budi Santoso"),
		BirthDate: &birthDate,
	}
	issuer := domain.User{
		Id:     "user-pdf-002",
		Name:   strPtr("Dr. Dewi Lestari"),
		Number: strPtr("196805152008123456"),
	}

	content := seeder.SeedBuildDiplomaContent(holder, issuer, 7, rng)
	pngBytes, err := seeder.SeedRenderDiplomaPNG(content)
	require.NoError(t, err)

	pdfBytes, err := seeder.SeedRenderDiplomaPDF(content, pngBytes)
	require.NoError(t, err)
	require.NotEmpty(t, pdfBytes)

	assert.Equal(t, []byte{0x25, 0x50, 0x44, 0x46, 0x2D}, pdfBytes[:5], "PDF magic header")
}

func TestSeedBuildDiplomaText(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	birthDate := time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC)

	holder := domain.User{
		Id:        "user-txt-001",
		Name:      strPtr("Budi Santoso"),
		BirthDate: &birthDate,
	}
	issuer := domain.User{
		Id:     "user-txt-002",
		Name:   strPtr("Dr. Dewi Lestari"),
		Number: strPtr("196805152008123456"),
	}

	content := seeder.SeedBuildDiplomaContent(holder, issuer, 3, rng)
	text := seeder.SeedBuildDiplomaText(content)

	assert.NotEmpty(t, text)
	assert.Contains(t, text, "Budi Santoso")
	assert.Contains(t, text, "S.Ked.")
	assert.Contains(t, text, "Dr. Dewi Lestari")
}

func TestSeedBuildDiplomaExtractionIDs(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	birthDate := time.Date(1998, 5, 12, 0, 0, 0, 0, time.UTC)

	holder := domain.User{
		Id:        "user-ext-001",
		Name:      strPtr("Budi Santoso"),
		BirthDate: &birthDate,
	}
	issuer := domain.User{
		Id:     "user-ext-002",
		Name:   strPtr("Dr. Dewi Lestari"),
		Number: strPtr("196805152008123456"),
	}

	content := seeder.SeedBuildDiplomaContent(holder, issuer, 0, rng)
	ids := seeder.SeedBuildDiplomaExtractionIDs(content)

	assert.Len(t, ids, 12)

	typeSet := make(map[string]bool)
	for _, id := range ids {
		assert.NotEmpty(t, id.Type, "each extraction ID must have a type")
		assert.NotEmpty(t, id.Value, "each extraction ID must have a value")
		assert.False(t, typeSet[id.Type], "each extraction ID type must be unique")
		typeSet[id.Type] = true
	}
}

func TestSeedCredentialNames(t *testing.T) {
	names := seeder.SeedCredentialNames()
	assert.Len(t, names, 10)
}

func TestSeedCredentialNames_AllNonEmpty(t *testing.T) {
	names := seeder.SeedCredentialNames()
	for _, n := range names {
		assert.NotEmpty(t, n.Name)
		assert.NotEmpty(t, n.Short)
	}
}

func TestSeedCredentialShortToProgramStudi(t *testing.T) {
	m := seeder.SeedCredentialShortToProgramStudi()

	expected := map[string]string{
		"S.Kom":    "Teknik Informatika",
		"S.Ak":     "Akuntansi",
		"S.H.":     "Ilmu Hukum",
		"S.Ked.":   "Pendidikan Dokter",
		"S.T.":     "Teknik Sipil",
		"S.E.":     "Ilmu Ekonomi",
		"S.Psi.":   "Psikologi",
		"S.Pd.":    "Pendidikan Bahasa Indonesia",
		"S.I.Kom.": "Ilmu Komunikasi",
		"S.A.B.":   "Administrasi Bisnis",
	}

	assert.Len(t, m, 10)
	for short, expectedProdi := range expected {
		got, ok := m[short]
		assert.True(t, ok, "short %q should exist in map", short)
		assert.Equal(t, expectedProdi, got)
	}
}

func TestSeedCredentialShortToProgramStudi_UnknownShort(t *testing.T) {
	m := seeder.SeedCredentialShortToProgramStudi()
	assert.Empty(t, m["XX"])
	assert.Empty(t, m[""])
	assert.Empty(t, m["S.I"])
}

func strPtr(s string) *string {
	return &s
}
