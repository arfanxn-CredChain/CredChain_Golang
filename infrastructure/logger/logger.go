package logger

import (
	"CredChain_Golang/config"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ZapLoggerParams holds dependencies for creating a zap logger.
type ZapLoggerParams struct {
	fx.In
	Config *config.Config
}

// NewZapLogger creates a production zap logger with caller info and error-level stack traces.
// Config is mandatory (reserved for future logger configuration: log level, format, output).
func NewZapLogger(p ZapLoggerParams) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	// AddCaller enabled globally to track exact file:line origins
	// AddStacktrace explicitly enabled for ErrorLevel and above
	logger, err := cfg.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// Module provides the unified logger to the Fx dependency injection container
var Module = fx.Provide(NewZapLogger)
