package rolldice

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"pub-service/logger"
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Handler struct {
	Metrics  Metrics
	Producer sarama.AsyncProducer
	Topic    string
}

type Metrics struct {
	RollCount metric.Int64Counter
}

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

	rdr := &Request{}
	err := json.NewDecoder(r.Body).Decode(rdr)
	if err != nil {
		log.Error("failed to decode RollDiceRequest", zap.Error(err))
		span.SetStatus(otelcodes.Error, "failed to decode RollDiceRequest")
		span.RecordError(err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log.Debug("rolldice request", zap.Any("request", rdr))
	distribution, err := h.roll(ctx, rdr.Sides, rdr.Rolls)
	if err != nil {
		span.SetStatus(otelcodes.Error, "failed to roll dice")
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	resp := &Response{
		Rolls:        rdr.Rolls,
		Sides:        rdr.Sides,
		Distribution: distribution,
	}
	log.Info("rolldice response", zap.Any("response", resp))

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Error("failed to encode RollDiceResponse", zap.Error(err))
		span.SetStatus(otelcodes.Error, "failed to encode RollDiceResponse")
		span.RecordError(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}

	encoder, err := json.Marshal(resp)
	if err != nil {
		log.Error("failed to encode RollDiceResponse", zap.Error(err))
		span.SetStatus(otelcodes.Error, "failed to encode RollDiceResponse")
		span.RecordError(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.publishRoll(ctx, encoder)
}

func (h *Handler) roll(ctx context.Context, sides int8, rolls int8) (map[int8]int32, error) {
	ctx, span := tracer.Start(ctx, "roll")
	defer span.End()
	log := logger.FromCtx(ctx)
	if rolls < 1 || rolls > 100 {
		err := fmt.Errorf("number of rolls must be >=1 and <=100")
		log.Error("invalid input", zap.Error(err))
		span.SetStatus(otelcodes.Error, "number of rolls must be >=1 and <=100")
		span.RecordError(err)
		return nil, err
	}
	if sides < 2 || sides > 100 {
		err := fmt.Errorf("number of sides must be >=2 and <=100")
		log.Error("invalid input", zap.Error(err))
		span.SetStatus(otelcodes.Error, "number of sides must be >=2 and <=100")
		span.RecordError(err)
		return nil, err
	}
	distribution := make(map[int8]int32)
	for i := int8(0); i < rolls; i++ {
		roll := int8(rand.Intn(int(sides)) + 1)
		distribution[roll]++
	}
	span.SetStatus(otelcodes.Ok, "success")
	return distribution, nil
}

func (h *Handler) publishRoll(ctx context.Context, roll json.RawMessage) {
	log := logger.FromCtx(ctx)

	msg := sarama.ProducerMessage{
		Topic: h.Topic,
		Value: sarama.ByteEncoder(roll),
	}

	// Inject tracing info into message
	span := createProducerSpan(ctx, &msg)
	defer span.End()

	// Send message and handle response
	startTime := time.Now()
	select {
	case h.Producer.Input() <- &msg:
		log.Info("Message sent to Kafka", zap.Any("message", msg))
		select {
		case successMsg := <-h.Producer.Successes():
			span.SetAttributes(
				attribute.Bool("messaging.kafka.producer.success", true),
				attribute.Int("messaging.kafka.producer.duration_ms", int(time.Since(startTime).Milliseconds())),
				attribute.KeyValue(semconv.MessagingKafkaMessageOffset(int(successMsg.Offset))),
			)
			log.Info("Successfully wrote message.", zap.Int64("offset", successMsg.Offset), zap.Duration("duration", time.Since(startTime)))
		case errMsg := <-h.Producer.Errors():
			span.SetAttributes(
				attribute.Bool("messaging.kafka.producer.success", false),
				attribute.Int("messaging.kafka.producer.duration_ms", int(time.Since(startTime).Milliseconds())),
			)
			span.SetStatus(otelcodes.Error, errMsg.Err.Error())
			log.Error("Failed to write message.", zap.Error(errMsg.Err))
		case <-ctx.Done():
			span.SetAttributes(
				attribute.Bool("messaging.kafka.producer.success", false),
				attribute.Int("messaging.kafka.producer.duration_ms", int(time.Since(startTime).Milliseconds())),
			)
			span.SetStatus(otelcodes.Error, "Context cancelled: "+ctx.Err().Error())
			log.Warn("Context canceled before success message received.", zap.Error(ctx.Err()))
		}
	case <-ctx.Done():
		span.SetAttributes(
			attribute.Bool("messaging.kafka.producer.success", false),
			attribute.Int("messaging.kafka.producer.duration_ms", int(time.Since(startTime).Milliseconds())),
		)
		span.SetStatus(otelcodes.Error, "Failed to send: "+ctx.Err().Error())
		log.Error("Failed to send message to Kafka within context deadline.", zap.Error(ctx.Err()))
		return
	}
	return
}

func createProducerSpan(ctx context.Context, msg *sarama.ProducerMessage) trace.Span {
	spanContext, span := tracer.Start(
		ctx,
		fmt.Sprintf("%s publish", msg.Topic),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.PeerService("kafka"),
			semconv.NetworkTransportTCP,
			semconv.MessagingSystemKafka,
			semconv.MessagingDestinationName(msg.Topic),
			semconv.MessagingOperationPublish,
			semconv.MessagingKafkaDestinationPartition(int(msg.Partition)),
		),
	)

	carrier := propagation.MapCarrier{}
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(spanContext, carrier)

	for key, value := range carrier {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{Key: []byte(key), Value: []byte(value)})
	}

	return span
}
