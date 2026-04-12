package middleware

import (
	appI18n "CredChain_Golang/infrastructure/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func I18nMiddleware(bundle *i18n.Bundle) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		localizer := i18n.NewLocalizer(bundle, lang)
		appI18n.SetLocalizer(c, localizer)
		c.Next()
	}
}

// GetLocalizer is now a shim for appI18n.GetLocalizer to avoid breaking middleware package callers
func GetLocalizer(c *gin.Context) *i18n.Localizer {
	return appI18n.GetLocalizer(c)
}
