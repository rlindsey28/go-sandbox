package kafka

import (
	"pub-service/config"
	"pub-service/logger"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

var (
	ProtocolVersion = sarama.V3_6_0_0
)

func NewProducer(config *config.KafkaConfig) (sarama.AsyncProducer, error) {
	log := logger.Get()

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = ProtocolVersion
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.NoResponse

	producer, err := sarama.NewAsyncProducer(config.Brokers, nil)
	if err != nil {
		log.Error("failed to create producer", zap.Error(err))
		return nil, err
	}

	go func() {
		for err := range producer.Errors() {
			log.Error("failed to send message", zap.Error(err))
		}
	}()

	return producer, nil
}
