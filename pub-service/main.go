package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"pub-service/config"
	"pub-service/health"
	"pub-service/kafka"
	"pub-service/logger"
	"pub-service/rolldice"
	"pub-service/telemetry"
	"time"

	"github.com/gorilla/mux"
	"github.com/sethvargo/go-envconfig"
	"go.uber.org/zap"
)

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

	// Setup Kafka
	producer, err := kafka.NewProducer(conf.Kafka)
	if err != nil {
		zaplog.Panic("failed to setup kafka", zap.Error(err))
	}

	// Setup router
	router := mux.NewRouter()

	healthHandler := health.Handler{}
	router.HandleFunc("/health", healthHandler.HealthCheck).Methods("GET")

	rollHandler := rolldice.Handler{
		Metrics:  rolldice.Metrics{},
		Producer: producer,
		Topic:    conf.Kafka.Topic,
	}
	rollHandler.Metrics.InitMetrics()
	router.HandleFunc("/rolldice", rollHandler.RollDice).Methods("POST")

	zaplog.Debug("starting server", zap.String("service-name", conf.ServiceName), zap.String("port", conf.Port))
	srv := &http.Server{
		Addr:         conf.Port,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      router,
	}

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	//Waite for shutdown signal
	select {
	case err := <-srvErr:
		zaplog.Error("server error", zap.Error(err))
		return
	case <-ctx.Done():
		// Wait for CTRL+C
		zaplog.Info("shutting down server")
		stop()
	}

}
