package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_KEY_EMPTY", "")
	fb := "fallback"
	got := getEnv("TEST_KEY_EMPTY", &fb)
	assert.Equal(t, &fb, got)
}

func TestGetEnv_Set_ReturnsValue(t *testing.T) {
	t.Setenv("TEST_KEY_SET", "hello")
	got := getEnv("TEST_KEY_SET", nil)
	assert.NotNil(t, got)
	assert.Equal(t, "hello", *got)
}

func TestGetIntEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_INT_EMPTY", "")
	fb := 42
	got := getIntEnv("TEST_INT_EMPTY", &fb)
	assert.Equal(t, 42, *got)
}

func TestGetIntEnv_Valid(t *testing.T) {
	t.Setenv("TEST_INT_VALID", "99")
	got := getIntEnv("TEST_INT_VALID", nil)
	assert.Equal(t, 99, *got)
}

func TestGetIntEnv_Invalid_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_INT_INVALID", "not-a-number")
	fb := 7
	got := getIntEnv("TEST_INT_INVALID", &fb)
	assert.Equal(t, 7, *got)
}

func TestGetBoolEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_BOOL_EMPTY", "")
	fb := true
	got := getBoolEnv("TEST_BOOL_EMPTY", &fb)
	assert.True(t, *got)
}

func TestGetBoolEnv_True(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	got := getBoolEnv("TEST_BOOL_TRUE", nil)
	assert.True(t, *got)
}

func TestGetBoolEnv_Other_ReturnsFalse(t *testing.T) {
	t.Setenv("TEST_BOOL_OTHER", "yes")
	got := getBoolEnv("TEST_BOOL_OTHER", nil)
	assert.False(t, *got)
}

func TestGetStringSliceEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_SLICE_EMPTY", "")
	fb := []string{"a", "b"}
	got := getStringSliceEnv("TEST_SLICE_EMPTY", fb)
	assert.Equal(t, fb, got)
}

func TestGetStringSliceEnv_CommaSplit(t *testing.T) {
	t.Setenv("TEST_SLICE_SET", "x,y,z")
	got := getStringSliceEnv("TEST_SLICE_SET", nil)
	assert.Equal(t, []string{"x", "y", "z"}, got)
}

func TestGetTimeEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_TIME_EMPTY", "")
	got := getTimeEnv("TEST_TIME_EMPTY", nil)
	assert.Nil(t, got)
}

func TestGetTimeEnv_Valid(t *testing.T) {
	t.Setenv("TEST_TIME_VALID", "1990-01-15")
	got := getTimeEnv("TEST_TIME_VALID", nil)
	assert.NotNil(t, got)
	assert.Equal(t, 1990, got.Year())
	assert.Equal(t, 15, got.Day())
}

func TestGetTimeEnv_Malformed_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_TIME_BAD", "not-a-date")
	got := getTimeEnv("TEST_TIME_BAD", nil)
	assert.Nil(t, got)
}

func TestGetJSONEnv_Empty_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_JSON_EMPTY", "")
	got := getJSONEnv("TEST_JSON_EMPTY", nil)
	assert.Nil(t, got)
}

func TestGetJSONEnv_Valid(t *testing.T) {
	t.Setenv("TEST_JSON_VALID", `{"key":"value","num":42}`)
	got := getJSONEnv("TEST_JSON_VALID", nil)
	assert.Equal(t, "value", got["key"])
	assert.Equal(t, float64(42), got["num"])
}

func TestGetJSONEnv_Invalid_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_JSON_INVALID", "not-json")
	got := getJSONEnv("TEST_JSON_INVALID", nil)
	assert.Nil(t, got)
}

func TestNewConfig_MissingJWTSecret_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("WALLET_ENCRYPTION_KEY", "some-key")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
}

func TestNewConfig_MissingWalletKey_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
}

func TestNewConfig_BothSet_ReturnsConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	cfg, err := NewConfig(".env.nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, "my-secret", *cfg.JWTSecret)
	assert.Equal(t, "my-wallet-key-exactly-32-chars-x", *cfg.WalletEncryptionKey)
}

func TestNewConfig_WalletKeyTooShort_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "too-short")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestNewConfig_WalletKeyTooLong_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "this-is-a-64-char-hex-encoded-key-that-should-fail-validation-x")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}
