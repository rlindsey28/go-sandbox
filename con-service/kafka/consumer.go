package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/rlindsey28/con-service/config"
	"github.com/rlindsey28/con-service/logger"
	"github.com/rlindsey28/con-service/rolldice"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

func NewConsumer(conf *config.KafkaConfig) {
	keepRunning := true
	log := logger.Get()
	log.Info("Starting a new Sarama consumer")

	/**
	 * Setup a new Sarama consumer group
	 */
	consumer := Consumer{
		ready: make(chan bool),
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := sarama.NewConsumerGroup(conf.Brokers, conf.ConsumerGroup, nil)
	if err != nil {
		log.Panic("Error creating consumer group client: %v", zap.Error(err))
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, strings.Split(conf.Topic, ","), &consumer); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}
				log.Panic("Error from consumer", zap.Error(err))
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	log.Info("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	for keepRunning {
		select {
		case <-ctx.Done():
			log.Info("terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Info("terminating: via signal")
			keepRunning = false
		}
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		log.Panic("Error closing client", zap.Error(err))
	}
}

type Consumer struct {
	ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log := logger.Get()
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/IBM/sarama/blob/main/consumer_group.go#L27-L29
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				log.Info("message channel was closed")
				return nil
			}
			log.Info("Message claimed", zap.Any("value", message.Value), zap.Time("timestamp", message.Timestamp))
			session.MarkMessage(message, "")

			roll := &rolldice.DiceRoll{}

			err := json.Unmarshal(message.Value, roll)
			if err != nil {
				log.Info("Error unmarshalling message", zap.Error(err))
			} else {
				log.Info("Dice roll", zap.Any("roll", roll))
			}
		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}
