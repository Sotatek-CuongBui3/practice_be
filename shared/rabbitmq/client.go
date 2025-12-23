package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Config holds RabbitMQ connection configuration
type Config struct {
	Host               string
	Port               int
	User               string
	Password           string
	VHost              string
	ExchangeName       string
	ExchangeType       string
	ExchangeDurable    bool
	ExchangeAutoDelete bool
	QueueName          string
	QueueDurable       bool
	QueueAutoDelete    bool
	QueueExclusive     bool
	RoutingKey         string
	RetryAttempts      int
	RetryInterval      time.Duration
	Heartbeat          time.Duration
	ConnectionTimeout  time.Duration
	PublishRetries     int
	PublishRetryDelay  time.Duration
	PublishBackoffMult float64
}

// Client represents a RabbitMQ client
type Client struct {
	config      *Config
	conn        *amqp.Connection
	channel     *amqp.Channel
	logger      *slog.Logger
	closeChan   chan *amqp.Error
	isConnected bool
}

// NewClient creates a new RabbitMQ client
func NewClient(config *Config, logger *slog.Logger) (*Client, error) {
	client := &Client{
		config:      config,
		logger:      logger,
		closeChan:   make(chan *amqp.Error),
		isConnected: false,
	}

	if err := client.connect(); err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	return client, nil
}

// connect establishes connection to RabbitMQ with retry logic
func (c *Client) connect() error {
	var err error

	dsn := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		c.config.User,
		c.config.Password,
		c.config.Host,
		c.config.Port,
		c.config.VHost,
	)

	amqpConfig := amqp.Config{
		Heartbeat: c.config.Heartbeat,
		Locale:    "en_US",
	}

	for attempt := 1; attempt <= c.config.RetryAttempts; attempt++ {
		c.logger.Info("Connecting to RabbitMQ",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", c.config.RetryAttempts),
		)

		c.conn, err = amqp.DialConfig(dsn, amqpConfig)
		if err == nil {
			c.logger.Info("Successfully connected to RabbitMQ")
			break
		}

		c.logger.Error("Failed to connect to RabbitMQ",
			slog.Any("error", err),
			slog.Int("attempt", attempt),
		)

		if attempt < c.config.RetryAttempts {
			time.Sleep(c.config.RetryInterval)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", c.config.RetryAttempts, err)
	}

	// Create channel
	c.channel, err = c.conn.Channel()
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to create channel: %w", err)
	}

	// Setup exchange and queue
	if err := c.setup(); err != nil {
		c.channel.Close()
		c.conn.Close()
		return fmt.Errorf("failed to setup exchange and queue: %w", err)
	}

	// Monitor connection
	c.closeChan = make(chan *amqp.Error)
	c.channel.NotifyClose(c.closeChan)
	c.isConnected = true

	c.logger.Info("RabbitMQ client initialized",
		slog.String("exchange", c.config.ExchangeName),
		slog.String("queue", c.config.QueueName),
	)

	return nil
}

// setup declares exchange, queue, and bindings
func (c *Client) setup() error {
	// Declare exchange
	err := c.channel.ExchangeDeclare(
		c.config.ExchangeName,       // name
		c.config.ExchangeType,       // type
		c.config.ExchangeDurable,    // durable
		c.config.ExchangeAutoDelete, // auto-deleted
		false,                       // internal
		false,                       // no-wait
		nil,                         // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err = c.channel.QueueDeclare(
		c.config.QueueName,       // name
		c.config.QueueDurable,    // durable
		c.config.QueueAutoDelete, // auto-delete
		c.config.QueueExclusive,  // exclusive
		false,                    // no-wait
		nil,                      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = c.channel.QueueBind(
		c.config.QueueName,    // queue name
		c.config.RoutingKey,   // routing key
		c.config.ExchangeName, // exchange
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	return nil
}

// Publish publishes a message to RabbitMQ
func (c *Client) Publish(ctx context.Context, body []byte, contentType string) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	err := c.channel.PublishWithContext(
		ctx,
		c.config.ExchangeName, // exchange
		c.config.RoutingKey,   // routing key
		false,                 // mandatory
		false,                 // immediate
		amqp.Publishing{
			ContentType:  contentType,
			Body:         body,
			DeliveryMode: amqp.Persistent, // persistent
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		c.logger.Error("Failed to publish message to RabbitMQ",
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	c.logger.Debug("Message published to RabbitMQ",
		slog.Int("body_size", len(body)),
		slog.String("content_type", contentType),
	)

	return nil
}

// Consume starts consuming messages from the queue
func (c *Client) Consume(consumerTag string) (<-chan amqp.Delivery, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("not connected to RabbitMQ")
	}

	messages, err := c.channel.Consume(
		c.config.QueueName, // queue
		consumerTag,        // consumer tag
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume messages: %w", err)
	}

	c.logger.Info("Started consuming messages from RabbitMQ",
		slog.String("queue", c.config.QueueName),
		slog.String("consumer_tag", consumerTag),
	)

	return messages, nil
}

// Close closes the RabbitMQ connection
func (c *Client) Close() error {
	c.logger.Info("Closing RabbitMQ connection")

	c.isConnected = false

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			c.logger.Error("Failed to close RabbitMQ channel",
				slog.Any("error", err),
			)
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.logger.Error("Failed to close RabbitMQ connection",
				slog.Any("error", err),
			)
			return err
		}
	}

	c.logger.Info("RabbitMQ connection closed successfully")
	return nil
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	return c.isConnected && c.conn != nil && !c.conn.IsClosed()
}

// GetChannel returns the channel for advanced operations
func (c *Client) GetChannel() *amqp.Channel {
	return c.channel
}

// PublishWithRetry publishes a message to RabbitMQ with retry logic and exponential backoff
func (c *Client) PublishWithRetry(ctx context.Context, body []byte, contentType string) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	maxRetries := c.config.PublishRetries
	if maxRetries <= 0 {
		maxRetries = 3 // default
	}

	baseDelay := c.config.PublishRetryDelay
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond // default
	}

	backoffMult := c.config.PublishBackoffMult
	if backoffMult <= 0 {
		backoffMult = 2.0 // default
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := c.channel.PublishWithContext(
			ctx,
			c.config.ExchangeName, // exchange
			c.config.RoutingKey,   // routing key
			false,                 // mandatory
			false,                 // immediate
			amqp.Publishing{
				ContentType:  contentType,
				Body:         body,
				DeliveryMode: amqp.Persistent, // persistent
				Timestamp:    time.Now(),
			},
		)

		if err == nil {
			if attempt > 0 {
				c.logger.Info("Successfully published message to RabbitMQ after retry",
					slog.Int("attempt", attempt+1),
					slog.Int("body_size", len(body)),
				)
			} else {
				c.logger.Debug("Message published to RabbitMQ",
					slog.Int("body_size", len(body)),
					slog.String("content_type", contentType),
				)
			}
			return nil
		}

		lastErr = err

		if attempt < maxRetries {
			// Calculate exponential backoff delay
			backoffDelay := time.Duration(float64(baseDelay) * float64(uint(1)<<uint(attempt)))
			c.logger.Warn("Failed to publish message to RabbitMQ, retrying...",
				slog.Int("attempt", attempt+1),
				slog.Int("max_retries", maxRetries),
				slog.Duration("retry_after", backoffDelay),
				slog.Any("error", err),
			)
			time.Sleep(backoffDelay)
		}
	}

	c.logger.Error("Failed to publish message to RabbitMQ after all retries",
		slog.Int("attempts", maxRetries+1),
		slog.Any("error", lastErr),
	)
	return fmt.Errorf("failed to publish message after %d attempts: %w", maxRetries+1, lastErr)
}
