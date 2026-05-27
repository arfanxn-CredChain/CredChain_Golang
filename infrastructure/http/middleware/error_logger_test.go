package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newObservedLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.ErrorLevel)
	return zap.New(core), logs
}

func TestErrorLoggerMiddleware_NoErrors_NoLog(t *testing.T) {
	logger, logs := newObservedLogger()
	mw := gin.HandlerFunc(NewErrorLoggerMiddleware(ErrorLoggerParams{Logger: logger}))

	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.Use(mw)
	engine.GET("/", func(c *gin.Context) { c.Status(200) })

	req, _ := http.NewRequest("GET", "/", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, 0, logs.Len())
}

func TestErrorLoggerMiddleware_OneError_OneLog(t *testing.T) {
	logger, logs := newObservedLogger()
	mw := gin.HandlerFunc(NewErrorLoggerMiddleware(ErrorLoggerParams{Logger: logger}))

	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.Use(mw)
	engine.GET("/test", func(c *gin.Context) {
		_ = c.Error(errors.New("something broke"))
		c.Status(500)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, 1, logs.Len())
	entry := logs.All()[0]
	assert.Equal(t, "request error", entry.Message)
	assert.Equal(t, "GET", entry.ContextMap()["method"])
	assert.Equal(t, "something broke", entry.ContextMap()["error"])
}

func TestErrorLoggerMiddleware_MultipleErrors_MultipleLog(t *testing.T) {
	logger, logs := newObservedLogger()
	mw := gin.HandlerFunc(NewErrorLoggerMiddleware(ErrorLoggerParams{Logger: logger}))

	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.Use(mw)
	engine.GET("/multi", func(c *gin.Context) {
		_ = c.Error(errors.New("err1"))
		_ = c.Error(errors.New("err2"))
		c.Status(500)
	})

	req, _ := http.NewRequest("GET", "/multi", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, 2, logs.Len())
}
