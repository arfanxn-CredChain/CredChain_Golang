package gintest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	appI18n "CredChain_Golang/infrastructure/i18n"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type contextConfig struct {
	user     *domain.User
	bundle   *i18n.Bundle
	method   string
	path     string
	body     any
	headers  map[string]string
	lang     string
	queryStr string
}

type ContextOption func(*contextConfig)

func WithUser(u *domain.User) ContextOption        { return func(c *contextConfig) { c.user = u } }
func WithI18nBundle(b *i18n.Bundle) ContextOption  { return func(c *contextConfig) { c.bundle = b } }
func WithMethod(m string) ContextOption            { return func(c *contextConfig) { c.method = m } }
func WithPath(p string) ContextOption              { return func(c *contextConfig) { c.path = p } }
func WithBody(v any) ContextOption                 { return func(c *contextConfig) { c.body = v } }
func WithAcceptLanguage(lang string) ContextOption { return func(c *contextConfig) { c.lang = lang } }
func WithQueryString(q string) ContextOption       { return func(c *contextConfig) { c.queryStr = q } }
func WithHeader(k, v string) ContextOption {
	return func(c *contextConfig) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[k] = v
	}
}

// NewContext builds a *gin.Context bound to a fresh httptest.ResponseRecorder.
func NewContext(t *testing.T, opts ...ContextOption) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	cfg := &contextConfig{method: http.MethodGet, path: "/"}
	for _, opt := range opts {
		opt(cfg)
	}

	var bodyBytes []byte
	if cfg.body != nil {
		var err error
		bodyBytes, err = json.Marshal(cfg.body)
		if err != nil {
			t.Fatalf("gintest.NewContext: marshal body: %v", err)
		}
	}

	url := cfg.path
	if cfg.queryStr != "" {
		url = cfg.path + "?" + cfg.queryStr
	}

	req := httptest.NewRequest(cfg.method, url, bytes.NewReader(bodyBytes))
	if cfg.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cfg.lang != "" {
		req.Header.Set("Accept-Language", cfg.lang)
	}
	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	if cfg.bundle != nil {
		lang := cfg.lang
		if lang == "" {
			lang = "id"
		}
		localizer := i18n.NewLocalizer(cfg.bundle, lang)
		appI18n.SetI18nLocalizer(c, localizer)
	}

	if cfg.user != nil {
		ctx := context.WithValue(req.Context(), httpContext.UserKey, cfg.user)
		c.Request = req.WithContext(ctx)
		c.Set(string(httpContext.UserKey), cfg.user)
	}

	return c, w
}
