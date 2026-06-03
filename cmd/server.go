package cmd

import (
	"context"

	"CredChain_Golang/domain"
	"CredChain_Golang/feature/auth"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/ai/pyai"
	"CredChain_Golang/infrastructure/chain"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	apphttp "CredChain_Golang/infrastructure/http"
	"CredChain_Golang/infrastructure/http/middleware"
	"CredChain_Golang/infrastructure/i18n"
	infraJobs "CredChain_Golang/infrastructure/jobs"
	infraLogger "CredChain_Golang/infrastructure/logger"
	"CredChain_Golang/infrastructure/oauth"
	"CredChain_Golang/infrastructure/storage"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the CredChain Gin Web Server",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			infraLogger.Module,
			fx.Provide(
				NewConfigFromCmd(cmd),
				func(lc fx.Lifecycle) context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					lc.Append(fx.Hook{OnStop: func(ctx context.Context) error { cancel(); return nil }})
					return ctx
				},
				i18n.NewI18nBundle,
				gormInfra.NewGorm,
				storage.NewStorage,
				storage.NewIPFSClient,
				pyai.NewPythonAIClient,
				chain.NewClient,
				chain.NewAuthorityService,
				chain.NewRegistryService,
				oauth.NewGoogleOAuthClient,
				user.NewGormUserRepository,
				user.NewGormUserTokenRepository,
				credential.NewGormCredentialRepository,
				credential.NewGormCredentialExtractJobRepository,
				credential.NewCredentialPolicy,
				credential.NewCredentialService,
				credential.NewCredentialHandler,
				infraJobs.NewCredentialExtractWorker,
				func(db *gorm.DB) domain.UnitOfWork {
					return gormInfra.NewGormUnitOfWork(db,
						user.NewGormUserRepository,
						credential.NewGormCredentialRepository,
						user.NewGormUserTokenRepository,
						credential.NewGormCredentialExtractJobRepository,
					)
				},
				user.NewUserPolicy,
				middleware.NewAuthMiddleware,
				middleware.NewAdminRoleMiddleware,
				middleware.NewIssuerRoleMiddleware,
				middleware.NewSuperAdminRoleMiddleware,
				middleware.NewI18nMiddleware,
				middleware.NewErrorLoggerMiddleware,
				middleware.NewLoginRateLimitMiddleware,
				middleware.NewRefreshRateLimitMiddleware,
				middleware.NewLogoutRateLimitMiddleware,
				middleware.NewApiRateLimitMiddleware,
				auth.NewAuthService,
				auth.NewAuthHandler,
				user.NewUserService,
				user.NewUserHandler,
				apphttp.NewGinRouter,
			),
			fx.Invoke(apphttp.RegisterRoutes),
			fx.Invoke(func(lc fx.Lifecycle, ctx context.Context, w *infraJobs.CredentialExtractWorker) {
				lc.Append(fx.Hook{
					OnStart: func(context.Context) error {
						w.Start(ctx)
						return nil
					},
				})
			}),
		).Run()
	},
}
