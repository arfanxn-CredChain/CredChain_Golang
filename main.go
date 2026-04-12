package main

import (
	"os"

	"CredChain_Golang/cmd"

	"go.uber.org/zap"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logger, _ := zap.NewProduction()
		defer logger.Sync()
		logger.Fatal("error executing command", zap.Error(err))
		os.Exit(1)
	}
}
