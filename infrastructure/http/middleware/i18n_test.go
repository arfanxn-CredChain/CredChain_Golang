package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"CredChain_Golang/config"
	appI18n "CredChain_Golang/infrastructure/i18n"
	"CredChain_Golang/infrastructure/testutil/gintest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestI18nMiddleware_SetsLocalizerFromAcceptLanguage(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	mw := gin.HandlerFunc(NewI18nMiddleware(I18nMiddlewareParams{Bundle: bundle}))

	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.Use(mw)
	engine.GET("/", func(c *gin.Context) {
		assert.NotNil(t, appI18n.GetI18nLocalizer(c))
		c.Status(200)
	})

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "en")
	engine.ServeHTTP(w, req)
}

func TestI18nMiddleware_MissingHeader_UsesDefault(t *testing.T) {
	bundle := gintest.LoadTestI18nBundle(t)
	mw := gin.HandlerFunc(NewI18nMiddleware(I18nMiddlewareParams{Bundle: bundle}))

	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.Use(mw)
	engine.GET("/", func(c *gin.Context) {
		assert.NotNil(t, appI18n.GetI18nLocalizer(c), "localizer must be set even without Accept-Language")
		c.Status(200)
	})

	req, _ := http.NewRequest("GET", "/", nil)
	engine.ServeHTTP(w, req)
}

func TestI18nMiddleware_IdLanguage(t *testing.T) {
	dir := "../../../locales"
	cfg := &config.Config{I18nLocalesDir: &dir}
	bundle, err := appI18n.NewI18nBundle(appI18n.I18nBundleParams{Config: cfg})
	assert.NoError(t, err)

	mw := gin.HandlerFunc(NewI18nMiddleware(I18nMiddlewareParams{Bundle: bundle}))
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.Use(mw)
	engine.GET("/", func(c *gin.Context) {
		assert.NotNil(t, appI18n.GetI18nLocalizer(c))
		c.Status(200)
	})

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "id")
	engine.ServeHTTP(w, req)
}
