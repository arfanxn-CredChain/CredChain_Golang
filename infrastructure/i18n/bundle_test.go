package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"CredChain_Golang/config"

	"github.com/gin-gonic/gin"
	goI18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"
)

func TestNewI18nBundle_LoadsLocales(t *testing.T) {
	dir := "../../locales"
	cfg := &config.Config{I18nLocalesDir: &dir}
	bundle, err := NewI18nBundle(I18nBundleParams{Config: cfg})
	assert.NoError(t, err)
	assert.NotNil(t, bundle)
}

func TestNewI18nBundle_MissingDir_ReturnsError(t *testing.T) {
	dir := "/nonexistent/path/to/locales"
	cfg := &config.Config{I18nLocalesDir: &dir}
	_, err := NewI18nBundle(I18nBundleParams{Config: cfg})
	assert.Error(t, err)
}

func TestGetSetI18nLocalizer_RoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	dir := "../../locales"
	cfg := &config.Config{I18nLocalesDir: &dir}
	bundle, _ := NewI18nBundle(I18nBundleParams{Config: cfg})

	localizer := goI18n.NewLocalizer(bundle, "en")
	SetI18nLocalizer(c, localizer)

	got := GetI18nLocalizer(c)
	assert.NotNil(t, got)
	assert.Same(t, localizer, got)
}

func TestGetI18nLocalizer_NotSet_ReturnsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	assert.Nil(t, GetI18nLocalizer(c))
}
