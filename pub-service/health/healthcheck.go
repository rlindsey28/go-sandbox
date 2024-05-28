package health

import (
	"encoding/json"
	"go-sandbox/logger"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type Handler struct{}

type Response struct {
	Status string `json:"status"`
}

const name = "healthcheck"

var (
	tracer = otel.Tracer(name)
)

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), name)
	defer span.End()

	resp := Response{Status: "OK"}
	log := logger.Get()
	log.Debug("health check", zap.String("status", resp.Status))
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Error("failed to encode response", zap.Error(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
