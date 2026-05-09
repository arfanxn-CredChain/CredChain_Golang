package logger

import (
	"CredChain_Golang/config"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ZapLoggerParams holds dependencies for creating a zap logger.
type ZapLoggerParams struct {
	fx.In
	Config *config.Config
}

// NewZapLogger creates a production zap logger with caller info and error-level stack traces.
// Config controls log level (LOG_LEVEL: debug/info/warn/error, default info) and
// output destination (LOG_OUTPUT: stdout or file path, default stdout).
func NewZapLogger(p ZapLoggerParams) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	// Set log level
	switch strings.ToLower(*p.Config.LogLevel) {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Set output
	if *p.Config.LogOutput != "stdout" {
		cfg.OutputPaths = []string{*p.Config.LogOutput}
		cfg.ErrorOutputPaths = []string{*p.Config.LogOutput, "stderr"}
	}

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
