package http

import (
	"context"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/auth"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/http/middleware"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewGinRouter() *gin.Engine {
	r := gin.Default()
	r.Use(cors.Default())
	return r
}

type RouteParams struct {
	fx.In

	Lifecycle              fx.Lifecycle
	Router                 *gin.Engine
	Config                 *config.Config
	Bundle                 *i18n.Bundle
	AuthHandler            *auth.Handler
	UserHandler            *user.Handler
	CredHandler            *credential.Handler
	Logger                 *zap.Logger
	AuthMiddleware         gin.HandlerFunc
	AdminRoleMiddleware    middleware.AdminRoleMiddleware
	IssuerRoleMiddleware   middleware.IssuerRoleMiddleware
	SuperAdminRoleMiddleware middleware.SuperAdminRoleMiddleware
}

// RegisterRoutes binds controllers and hooks the Gin start/stop lifecycle
func RegisterRoutes(p RouteParams) {
	// Global middleware — order matters:
	// 1. ErrorLogger runs last (collects c.Error() entries after all handlers complete)
	// 2. I18nMiddleware runs first so localizers are available to all handlers
	p.Router.Use(middleware.ErrorLogger(p.Logger))
	p.Router.Use(middleware.I18nMiddleware(p.Bundle))

	// All API routes namespaced under /api
	api := p.Router.Group("/api")
	{
		// Open routes
		api.GET("/health", func(c *gin.Context) {
			responder.Send(c, domain.CodeSystemSuccess, gin.H{"status": "ok"})
		})
		api.POST("/auth/google", p.AuthHandler.HandleGoogleLogin)

		// Secure routes
		secure := api.Group("/")
		secure.Use(p.AuthMiddleware)
		{
			// Users API
			users := secure.Group("/users")
			{
				users.GET("", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Paginate)
				users.GET("/self", p.UserHandler.Find)
				users.GET("/self/credentials", p.UserHandler.GetSelfCredentials)
				users.PUT("/self/profile", p.UserHandler.UpdateSelfProfile)
				users.PUT("/self/email", p.UserHandler.UpdateSelfEmail)
				users.GET("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.FindByAdmin)
				users.POST("/batch", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Store)
				users.PUT("/batch/role", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.BatchUpdateRole)
			}

			// Credentials API
			creds := secure.Group("/credentials")
			{
				creds.GET("", gin.HandlerFunc(p.AdminRoleMiddleware), p.CredHandler.GetCredentials)
				creds.GET("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.CredHandler.GetCredentialByID)
				creds.POST("/batch/issue", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredHandler.IssueCredential)
				creds.POST("/batch/revoke", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredHandler.RevokeCredential)
				creds.POST("/verify", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredHandler.VerifyHash)
			}
		}
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				p.Logger.Info("credchain golang backend starting", zap.String("port", p.Config.AppPort))
				if err := p.Router.Run(":" + p.Config.AppPort); err != nil {
					p.Logger.Error("server failed to start", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			p.Logger.Info("stopping server")
			return nil
		},
	})
}
