package seeder_test

import (
	"testing"

	"CredChain_Golang/infrastructure/database/seeder"
	"github.com/stretchr/testify/assert"
)

func TestSanitizePhone_AlreadyValidE164(t *testing.T) {
	result := seeder.SanitizePhone("+6289506089254")
	assert.Equal(t, "+6289506089254", result)
}

func TestSanitizePhone_StripsWhitespace(t *testing.T) {
	result := seeder.SanitizePhone("+62 895 0608 9254")
	assert.Equal(t, "+6289506089254", result)
}

func TestSanitizePhone_StripsAllWhitespace(t *testing.T) {
	result := seeder.SanitizePhone("  +62  852  2829  6172  ")
	assert.Equal(t, "+6285228296172", result)
}

func TestSanitizePhone_IndonesianPrefix08_BecomesPlus62(t *testing.T) {
	result := seeder.SanitizePhone("089506089254")
	assert.Equal(t, "+6289506089254", result)
}

func TestSanitizePhone_IndonesianPrefix08_WithSpaces(t *testing.T) {
	result := seeder.SanitizePhone("08 522 829 6172")
	assert.Equal(t, "+6285228296172", result)
}

func TestSanitizePhone_EmptyString(t *testing.T) {
	result := seeder.SanitizePhone("")
	assert.Equal(t, "", result)
}

func TestSanitizePhone_OnlyWhitespace(t *testing.T) {
	result := seeder.SanitizePhone("   ")
	assert.Equal(t, "", result)
}
