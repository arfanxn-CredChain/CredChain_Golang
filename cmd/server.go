package cmd

import (
	"fmt"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/auth"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/ai"
	"CredChain_Golang/infrastructure/chain"
	"CredChain_Golang/infrastructure/database"
	apphttp "CredChain_Golang/infrastructure/http"
	"CredChain_Golang/infrastructure/i18n"
	applogger "CredChain_Golang/infrastructure/logger"
	"CredChain_Golang/infrastructure/storage"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

func checkSystemInitialized(db *database.DB, cfg *config.Config, logger *zap.Logger) error {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM users WHERE role = $1", domain.RoleSuperAdmin)
	if err != nil {
		return fmt.Errorf("failed to verify system initialization state: %v", err)
	}

	if count == 0 {
		logger.Fatal("system is not fully initialized. super admin is missing. please run `go run cmd/server/main.go init-super-admin` before starting the server.")
		return fmt.Errorf("system missing super admin")
	}

	logger.Info("system initialization verified")
	return nil
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the CredChain Gin Web Server",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			applogger.Module,
			fx.Provide(
				config.LoadConfig,
				i18n.NewBundle,
				database.ConnectPostgres,
				database.ConnectMongo,
				storage.NewStorage,
				storage.NewIPFSClient,
				ai.NewClient,
				chain.NewClient,
				user.NewRepository,
				credential.NewRepository,
				auth.NewHandler,
				user.NewService,
				user.NewHandler,
				credential.NewService,
				credential.NewHandler,
				apphttp.NewGinRouter,
			),
			fx.Invoke(checkSystemInitialized),
			fx.Invoke(apphttp.RegisterRoutes),
		).Run()
	},
}

