package rolldice

import (
	"context"
	"encoding/json"
	"fmt"
	"go-sandbox/logger"
	"math/rand"
	"net/http"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Handler struct{}

type Request struct {
	Sides int8 `json:"sides"`
	Rolls int8 `json:"rolls"`
}

type Response struct {
	Rolls        int8           `json:"rolls"`
	Sides        int8           `json:"sides"`
	Distribution map[int8]int32 `json:"distribution"`
}

const name = "rolldice"

var (
	tracer  = otel.Tracer(name)
	meter   = otel.Meter(name)
	rollCnt metric.Int64Counter
)

func init() {
	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of rolls by roll value"),
		metric.WithUnit("{roll}"))
	if err != nil {
		panic(err)
	}
}

func RegisterRoutes(router *mux.Router) {
	h := Handler{}

	router.HandleFunc("/rolldice", h.rollDice).Methods("POST")
}

func (h *Handler) rollDice(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "rollDice")
	defer span.End()

	log := logger.Get()
	rdr := &Request{}
	err := json.NewDecoder(r.Body).Decode(rdr)
	if err != nil {
		log.Error("failed to decode RollDiceRequest", zap.Error(err))
		span.SetStatus(codes.Error, "failed to decode RollDiceRequest")
		span.RecordError(err)
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	log.Debug("rolldice request", zap.Any("request", rdr))
	distribution, err := roll(ctx, rdr.Sides, rdr.Rolls)
	if err != nil {
		span.SetStatus(codes.Error, "failed to roll dice")
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := &Response{
		Rolls:        rdr.Rolls,
		Sides:        rdr.Sides,
		Distribution: distribution,
	}
	log.Debug("rolldice response", zap.Any("response", resp))
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Error("failed to encode RollDiceResponse", zap.Error(err))
		span.SetStatus(codes.Error, "failed to encode RollDiceResponse")
		span.RecordError(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
	span.SetStatus(codes.Ok, "success")
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
