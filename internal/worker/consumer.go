package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cuongbtq/practice-be/internal/worker/domain"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// setupConsumer sets up RabbitMQ consumer with QoS and returns delivery channel
func (w *Worker) setupConsumer(ctx context.Context) (<-chan amqp.Delivery, error) {
	// Get the RabbitMQ channel
	channel := w.rabbitClient.GetChannel()
	if channel == nil {
		return nil, fmt.Errorf("rabbitmq channel is nil")
	}

	// Set QoS (Quality of Service) to control message prefetching
	// prefetch_count: number of unacknowledged messages per consumer
	// prefetch_size: 0 means no specific byte limit
	// global: false means per-consumer, not per-channel
	err := channel.Qos(
		w.prefetchCount, // prefetch count from config
		0,               // prefetch size
		false,           // global
	)

	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	w.logger.Info("RabbitMQ QoS configured",
		slog.Int("prefetch_count", w.prefetchCount),
	)

	// Create unique consumer tag using worker ID
	consumerTag := w.workerID

	// Start consuming messages from the queue
	// auto-ack: false (manual acknowledgment for reliability)
	// exclusive: false (allow multiple consumers)
	deliveries, err := w.rabbitClient.Consume(consumerTag)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	w.logger.Info("RabbitMQ consumer started",
		slog.String("consumer_tag", consumerTag),
		slog.String("worker_id", w.workerID),
		slog.String("queue", w.rabbitMQQueueName),
	)

	return deliveries, nil
}

// startMessageDispatcher listens to RabbitMQ deliveries and dispatches jobs to worker pool
func (w *Worker) startMessageDispatcher(ctx context.Context, deliveries <-chan amqp.Delivery) {
	w.logger.Info("Message dispatcher started",
		slog.String("worker_id", w.workerID),
	)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Message dispatcher stopped - context canceled")
			return

		case delivery, ok := <-deliveries:
			if !ok {
				w.logger.Warn("RabbitMQ delivery channel closed")
				return
			}

			// Parse message body to extract job_id
			var msg struct {
				JobID string `json:"job_id"`
			}

			if err := json.Unmarshal(delivery.Body, &msg); err != nil {
				w.logger.Error("Failed to parse message JSON",
					slog.String("error", err.Error()),
					slog.String("body", string(delivery.Body)),
				)
				// NACK message without requeue - malformed messages should go to DLQ
				if nackErr := delivery.Nack(false, false); nackErr != nil {
					w.logger.Error("Failed to NACK malformed message",
						slog.String("error", nackErr.Error()),
					)
				}
				continue
			}

			// Validate job_id is a valid UUID
			if _, err := uuid.Parse(msg.JobID); err != nil {
				w.logger.Error("Invalid job_id format - not a UUID",
					slog.String("job_id", msg.JobID),
					slog.String("error", err.Error()),
				)
				// NACK message without requeue - invalid UUID
				if nackErr := delivery.Nack(false, false); nackErr != nil {
					w.logger.Error("Failed to NACK message with invalid job_id",
						slog.String("error", nackErr.Error()),
					)
				}
				continue
			}

			// Create JobMessage with job_id and delivery tag
			jobMsg := &domain.JobMessage{
				JobID:       msg.JobID,
				DeliveryTag: delivery.DeliveryTag,
			}

			// Send to worker pool via jobsChan
			select {
			case w.jobsChan <- jobMsg:
				w.logger.Debug("Job dispatched to worker pool",
					slog.String("job_id", msg.JobID),
					slog.Uint64("delivery_tag", delivery.DeliveryTag),
				)
			case <-ctx.Done():
				w.logger.Info("Message dispatcher stopped while dispatching job")
				// NACK the message so it can be reprocessed
				if nackErr := delivery.Nack(false, true); nackErr != nil {
					w.logger.Error("Failed to NACK message on shutdown",
						slog.String("error", nackErr.Error()),
					)
				}
				return
			}
		}
	}
}
