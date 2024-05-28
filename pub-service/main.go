package main

import (
	"context"
	"errors"
	"go-sandbox/config"
	"go-sandbox/health"
	"go-sandbox/logger"
	"go-sandbox/rolldice"
	"go-sandbox/telemetry"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
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

	// Setup otel
	otelShutdown, err := telemetry.SetupOtelSDK(ctx, conf)
	if err != nil {
		zaplog.Panic("failed to setup otel", zap.Error(err))
	}
	// Handle otel shutdown
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Setup router
	router := mux.NewRouter()

	healthHandler := health.Handler{}
	router.HandleFunc("/health", healthHandler.HealthCheck).Methods("GET")

	rollHandler := rolldice.Handler{
		Metrics: rolldice.Metrics{},
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
