package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// MinPort is the minimum valid port number
	MinPort = 1
	// MaxPort is the maximum valid port number
	MaxPort = 65535
)

// Config represents the complete application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
	Logging  LoggingConfig  `yaml:"logging"`
	App      AppConfig      `yaml:"app"`
	Worker   WorkerConfig   `yaml:"worker"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL connection configuration
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
}

// RabbitMQConfig holds RabbitMQ connection and exchange/queue configuration
type RabbitMQConfig struct {
	Host       string           `yaml:"host"`
	Port       int              `yaml:"port"`
	User       string           `yaml:"user"`
	Password   string           `yaml:"password"`
	VHost      string           `yaml:"vhost"`
	Exchange   ExchangeConfig   `yaml:"exchange"`
	Queue      QueueConfig      `yaml:"queue"`
	RoutingKey string           `yaml:"routing_key"`
	Connection ConnectionConfig `yaml:"connection"`
	Publish    PublishConfig    `yaml:"publish"`
	Consumer   ConsumerConfig   `yaml:"consumer"`
}

// ExchangeConfig holds RabbitMQ exchange configuration
type ExchangeConfig struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Durable    bool   `yaml:"durable"`
	AutoDelete bool   `yaml:"auto_delete"`
}

// QueueConfig holds RabbitMQ queue configuration
type QueueConfig struct {
	Name       string `yaml:"name"`
	Durable    bool   `yaml:"durable"`
	AutoDelete bool   `yaml:"auto_delete"`
	Exclusive  bool   `yaml:"exclusive"`
}

// ConnectionConfig holds RabbitMQ connection settings
type ConnectionConfig struct {
	RetryAttempts     int           `yaml:"retry_attempts"`
	RetryInterval     time.Duration `yaml:"retry_interval"`
	Heartbeat         time.Duration `yaml:"heartbeat"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
}

// PublishConfig holds RabbitMQ publish retry settings
type PublishConfig struct {
	RetryAttempts     int           `yaml:"retry_attempts"`
	RetryInterval     time.Duration `yaml:"retry_interval"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier"`
}

// ConsumerConfig holds RabbitMQ consumer settings
type ConsumerConfig struct {
	PrefetchCount int  `yaml:"prefetch_count"`
	AutoAck       bool `yaml:"auto_ack"`
	Exclusive     bool `yaml:"exclusive"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level            string `yaml:"level"`
	Format           string `yaml:"format"`
	Output           string `yaml:"output"`
	EnableCaller     bool   `yaml:"enable_caller"`
	EnableStackTrace bool   `yaml:"enable_stack_trace"`
}

// AppConfig holds application metadata
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
}

// WorkerConfig holds worker service configuration
type WorkerConfig struct {
	Concurrency       int           `yaml:"concurrency"`
	MaxJobs           int           `yaml:"max_jobs"`
	JobTimeout        time.Duration `yaml:"job_timeout"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
}

// Load reads and parses the configuration file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) ValidateAPIConfig() error {
	if c.Server.Port < MinPort || c.Server.Port > MaxPort {
		return fmt.Errorf("invalid server port: %d (must be between %d and %d)", c.Server.Port, MinPort, MaxPort)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Port < MinPort || c.Database.Port > MaxPort {
		return fmt.Errorf("invalid database port: %d (must be between %d and %d)", c.Database.Port, MinPort, MaxPort)
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.RabbitMQ.Host == "" {
		return fmt.Errorf("rabbitmq host is required")
	}

	if c.RabbitMQ.Port < MinPort || c.RabbitMQ.Port > MaxPort {
		return fmt.Errorf("invalid rabbitmq port: %d (must be between %d and %d)", c.RabbitMQ.Port, MinPort, MaxPort)
	}

	if c.RabbitMQ.Exchange.Name == "" {
		return fmt.Errorf("rabbitmq exchange name is required")
	}

	if c.RabbitMQ.Queue.Name == "" {
		return fmt.Errorf("rabbitmq queue name is required")
	}

	return nil
}

// Make another validation function for worker config
func (c *Config) ValidateWorkerConfig() error {
	if c.Worker.Concurrency <= 0 {
		return fmt.Errorf("worker concurrency must be greater than 0")
	}

	if c.Worker.MaxJobs <= 0 {
		return fmt.Errorf("worker max_jobs must be greater than 0")
	}

	if c.Worker.JobTimeout <= 0 {
		return fmt.Errorf("worker job_timeout must be greater than 0")
	}

	if c.Worker.HeartbeatInterval <= 0 {
		return fmt.Errorf("worker heartbeat_interval must be greater than 0")
	}

	if c.Worker.ShutdownTimeout <= 0 {
		return fmt.Errorf("worker shutdown_timeout must be greater than 0")
	}

	return nil
}
