package main

import (
	"context"
	"errors"
	"go-sandbox/config"
	"go-sandbox/health"
	"go-sandbox/logger"
	"go-sandbox/rolldice"
	"go-sandbox/telemetry"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func main() {
	// Handle SIGINT gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Load config
	appconfig := config.Initialize()
	log := logger.Get()

	// Setup otel
	otelShutdown, err := telemetry.SetupOtelSDK(ctx, *appconfig)
	if err != nil {
		log.Panic("failed to setup otel", zap.Error(err))
	}
	// Handle otel shutdown
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Setup router
	router := mux.NewRouter()
	health.RegisterRoutes(router)
	rolldice.RegisterRoutes(router)

	log.Debug("starting server", zap.String("service-name", appconfig.Service.Name), zap.String("port", appconfig.Service.Port))
	srv := &http.Server{
		Addr:         appconfig.Service.Port,
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
		log.Error("server error", zap.Error(err))
		return
	case <-ctx.Done():
		// Wait for CTRL+C
		log.Info("shutting down server")
		stop()
	}

}
