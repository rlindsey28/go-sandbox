package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

var once sync.Once

var logger *zap.Logger

// Get initializes a zap.Logger instance if it has not been initialized
// already and returns the same instance for subsequent calls.
func Get() *zap.Logger {
	once.Do(func() {
		stdout := zapcore.AddSync(os.Stdout)

		serviceLevel := zap.InfoLevel
		envLevel := os.Getenv("LOG_LEVEL")
		if envLevel != "" {
			levelFromEnv, err := zapcore.ParseLevel(envLevel)
			if err != nil {
				log.Println(
					fmt.Errorf("invalid envLevel, defaulting to INFO: %w", err),
				)
				levelFromEnv = zap.InfoLevel
			}
			serviceLevel = levelFromEnv
		}

		atomicLevel := zap.NewAtomicLevelAt(serviceLevel)
		developmentCfg := zap.NewDevelopmentEncoderConfig()
		developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)

		// log to console
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, stdout, atomicLevel),
		)

		logger = zap.New(core).With(zap.String("app", "pub-service"))
	})

	return logger
}

// FromCtx returns the Logger associated with the ctx. If no logger
// is associated, the default logger is returned, unless it is nil
// in which case a disabled logger is returned.
func FromCtx(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	} else if l := logger; l != nil {
		return l
	}

	return zap.NewNop()
}

// WithCtx returns a copy of ctx with the Logger attached.
func WithCtx(ctx context.Context, l *zap.Logger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		if lp == l {
			// Do not store same logger.
			return ctx
		}
	}

	return context.WithValue(ctx, ctxKey{}, l)
}
