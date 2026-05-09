package middleware

import (
	appI18n "CredChain_Golang/infrastructure/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/fx"
)

type I18nMiddleware gin.HandlerFunc

type I18nMiddlewareParams struct {
	fx.In
	Bundle *i18n.Bundle
}

func NewI18nMiddleware(p I18nMiddlewareParams) I18nMiddleware {
	return I18nMiddleware(func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		localizer := i18n.NewLocalizer(p.Bundle, lang)
		appI18n.SetI18nLocalizer(c, localizer)
		c.Next()
	})
}
