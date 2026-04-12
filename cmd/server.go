package cmd

import (
	"CredChain_Golang/config"
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
)

func init() {
	rootCmd.AddCommand(serverCmd)
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
			fx.Invoke(apphttp.RegisterRoutes),
		).Run()
	},
}
