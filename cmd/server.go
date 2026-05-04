package cmd

import (
	"context"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/auth"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/ai"
	"CredChain_Golang/infrastructure/chain"
	gormInfra "CredChain_Golang/infrastructure/gorm"
	apphttp "CredChain_Golang/infrastructure/http"
	"CredChain_Golang/infrastructure/http/middleware"
	"CredChain_Golang/infrastructure/i18n"
	applogger "CredChain_Golang/infrastructure/logger"
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
			applogger.Module,
			fx.Provide(
				func() *config.Config {
					return cmd.Context().Value(ConfigContextKey).(*config.Config)
				},
				func(lc fx.Lifecycle) context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					lc.Append(fx.Hook{
						OnStop: func(ctx context.Context) error {
							cancel()
							return nil
						},
					})
					return ctx
				},
				i18n.NewBundle,
				gormInfra.NewGorm,
				storage.NewStorage,
				storage.NewIPFSClient,
				ai.NewClient,
				chain.NewClient,
				oauth.NewGoogleOAuthClient,
				user.NewGormUserRepository,
				user.NewGormUserTokenRepository,
				credential.NewGormCredentialRepository,
				// UoW with repository factories
				func(db *gormInfra.GormDB) domain.UnitOfWork {
					return gormInfra.NewGormUnitOfWork(
						db,
						func(tx *gorm.DB) domain.UserRepository {
							return user.NewGormUserRepository(&gormInfra.GormDB{DB: tx})
						},
						func(tx *gorm.DB) domain.CredentialRepository {
							return credential.NewGormCredentialRepository(&gormInfra.GormDB{DB: tx})
						},
						func(tx *gorm.DB) domain.UserTokenRepository {
							return user.NewGormUserTokenRepository(user.UserTokenRepoParams{DB: tx})
						},
					)
				},
				user.NewUserPolicy,
				middleware.NewAuthMiddleware,
				middleware.NewAdminRoleMiddleware,
				middleware.NewIssuerRoleMiddleware,
				middleware.NewSuperAdminRoleMiddleware,
				auth.NewAuthService,
				auth.NewAuthHandler,
				user.NewUserService,
				user.NewUserHandler,
				credential.NewCredentialService,
				credential.NewCredentialHandler,
				apphttp.NewGinRouter,
			),
			fx.Invoke(apphttp.RegisterRoutes),
		).Run()
	},
}
