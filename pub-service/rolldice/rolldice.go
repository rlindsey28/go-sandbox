package rolldice

import (
	"context"
	"encoding/json"
	"fmt"
	"go-sandbox/kafka"
	"go-sandbox/logger"
	"math/rand"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Handler struct {
	Metrics   Metrics
	Publisher *kafka.Publisher
}

type Metrics struct {
	RollCount metric.Int64Counter
}

type RollDiceRequest struct {
	Sides int8 `json:"sides"`
	Rolls int8 `json:"rolls"`
}

type RollDiceResponse struct {
	Rolls        int8           `json:"rolls"`
	Sides        int8           `json:"sides"`
	Distribution map[int8]int32 `json:"distribution"`
}

const name = "rolldice"

var (
	tracer = otel.Tracer(name)
)

func (m *Metrics) InitMetrics() {
	log := logger.Get()
	meter := otel.Meter(name)

	var err error
	m.RollCount, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of API calls"),
		metric.WithUnit("{call}"))
	if err != nil {
		log.Error("failed to create counter", zap.Error(err))
	}
}

func (h *Handler) RollDice(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	ctx, span := tracer.Start(r.Context(), "rollDice")
	defer span.End()
	h.Metrics.RollCount.Add(ctx, 1)

	rdr := &RollDiceRequest{}
	err := json.NewDecoder(r.Body).Decode(rdr)
	if err != nil {
		log.Error("failed to decode RollDiceRequest", zap.Error(err))
		span.SetStatus(codes.Error, "failed to decode RollDiceRequest")
		span.RecordError(err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log.Debug("rolldice request", zap.Any("request", rdr))
	distribution, err := roll(ctx, rdr.Sides, rdr.Rolls)
	if err != nil {
		span.SetStatus(codes.Error, "failed to roll dice")
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	resp := &RollDiceResponse{
		Rolls:        rdr.Rolls,
		Sides:        rdr.Sides,
		Distribution: distribution,
	}
	log.Info("rolldice response", zap.Any("response", resp))
	encoder, err := json.Marshal(resp)
	if err != nil {
		log.Error("failed to encode RollDiceResponse", zap.Error(err))
		span.SetStatus(codes.Error, "failed to encode RollDiceResponse")
		span.RecordError(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = h.Publisher.Publish(ctx, encoder)
	if err != nil {
		log.Error("failed to publish response to kafka", zap.Error(err))
		span.SetStatus(codes.Error, "failed to publish response to kafka")
		span.RecordError(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Error("failed to encode RollDiceResponse", zap.Error(err))
		span.SetStatus(codes.Error, "failed to encode RollDiceResponse")
		span.RecordError(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func roll(ctx context.Context, sides int8, rolls int8) (map[int8]int32, error) {
	ctx, span := tracer.Start(ctx, "roll")
	defer span.End()
	log := logger.FromCtx(ctx)
	if rolls < 1 || rolls > 100 {
		err := fmt.Errorf("number of rolls must be >=1 and <=100")
		log.Error("invalid input", zap.Error(err))
		span.SetStatus(codes.Error, "number of rolls must be >=1 and <=100")
		span.RecordError(err)
		return nil, err
	}
	if sides < 2 || sides > 100 {
		err := fmt.Errorf("number of sides must be >=2 and <=100")
		log.Error("invalid input", zap.Error(err))
		span.SetStatus(codes.Error, "number of sides must be >=2 and <=100")
		span.RecordError(err)
		return nil, err
	}
	distribution := make(map[int8]int32)
	for i := int8(0); i < rolls; i++ {
		roll := int8(rand.Intn(int(sides)) + 1)
		distribution[roll]++
	}
	span.SetStatus(codes.Ok, "success")
	return distribution, nil
}
