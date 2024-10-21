package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/rlindsey28/con-service/config"
	"github.com/rlindsey28/con-service/kafka"
	"github.com/rlindsey28/con-service/logger"
	"github.com/rlindsey28/con-service/telemetry"
	"github.com/sethvargo/go-envconfig"
	"go.uber.org/zap"
)

type RollDiceResponse struct {
	Rolls        int8           `json:"rolls"`
	Sides        int8           `json:"sides"`
	Distribution map[int8]int32 `json:"distribution"`
}

func main() {
	// Handle SIGINT gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Load config
	var conf config.AppConfig
	if err := envconfig.Process(ctx, &conf); err != nil {
		log.Panic("failed to process config", zap.Error(err))
	}
	zaplog := logger.Get()

	zaplog.Info("loaded config", zap.Any("config", conf))
	// Setup otel
	otelShutdown, err := telemetry.SetupOtelSDK(ctx, conf)
	if err != nil {
		zaplog.Panic("failed to setup otel", zap.Error(err))
	}

	// Handle shutdown
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	kafka.NewConsumer(conf.Kafka)
	//Wait for shutdown signal
	select {
	case <-ctx.Done():
		// Wait for CTRL+C
		zaplog.Info("shutting down server")
		stop()
	}
}
