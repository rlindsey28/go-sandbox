package kafka

import (
	"context"
	"encoding/json"
	"go-sandbox/logger"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type Publisher struct {
	topicName string
	producer  sarama.SyncProducer
}

func NewPublisher(topicName string, producer sarama.SyncProducer) *Publisher {
	return &Publisher{
		topicName: topicName,
		producer:  producer,
	}
}

func (p *Publisher) Publish(ctx context.Context, message json.RawMessage) error {
	log := logger.Get()
	log.Info("Enter publish ===>>>>")

	defer log.Info("Finished publish ===>>>>")

	log.Info("Publishing message to kafka", zap.String("message", string(message)))
	_, _, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: p.topicName,
		Key:   sarama.StringEncoder("test"),
		Value: sarama.ByteEncoder(message),
	})
	if err != nil {
		log.Error("failed to publish event to kafka: %v", zap.Error(err))
		return err
	}

	return nil

}
