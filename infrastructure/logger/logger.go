package logger

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	
	// AddCaller enabled globally to track exact file:line origins
	// AddStacktrace explicitly enabled for ErrorLevel and above
	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// Module provides the unified logger to the Fx dependency injection container
var Module = fx.Provide(NewLogger)
