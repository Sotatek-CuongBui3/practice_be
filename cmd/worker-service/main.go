package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cuongbtq/practice-be/internal/config"
	"github.com/cuongbtq/practice-be/internal/worker"
	"github.com/cuongbtq/practice-be/shared/logger"
	"github.com/cuongbtq/practice-be/shared/postgresql"
	"github.com/cuongbtq/practice-be/shared/rabbitmq"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables or flags")
	}

	// Parse command-line flags
	defaultConfigPath := os.Getenv("WORKER_SERVICE_CONFIG_PATH")
	if defaultConfigPath == "" {
		defaultConfigPath = "configs/worker-service/config.yaml"
	}
	configPath := flag.String("config", defaultConfigPath, "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Initialize logger
	appLogger, err := initLogger(&cfg.Logging)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	appLogger.Info("Starting worker service",
		slog.String("app", cfg.App.Name),
		slog.String("version", cfg.App.Version),
		slog.String("environment", cfg.App.Environment),
	)

	// Initialize PostgreSQL client
	dbClient, err := initPostgreSQL(&cfg.Database, appLogger.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	appLogger.Info("Database connection established")

	// Initialize RabbitMQ client
	rabbitClient, err := initRabbitMQ(&cfg.RabbitMQ, appLogger.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize RabbitMQ: %w", err)
	}

	appLogger.Info("RabbitMQ connection established")

	// Create worker instance
	workerInstance := worker.NewWorker(&worker.Config{
		Logger:       appLogger.Logger,
		DBClient:     dbClient,
		RabbitClient: rabbitClient,
		Concurrency:  cfg.Worker.Concurrency,
		JobTimeout:   cfg.Worker.JobTimeout,
	})

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := workerInstance.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	appLogger.Info("Worker service started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		appLogger.Info("Received signal, shutting down gracefully",
			slog.String("signal", sig.String()),
		)
	case err := <-errChan:
		appLogger.Error("Worker error",
			slog.Any("error", err),
		)
		return err
	}

	// Cancel context to stop worker
	cancel()

	// Give worker time to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop worker
	done := make(chan struct{})
	go func() {
		workerInstance.Stop()
		close(done)
	}()

	select {
	case <-done:
		appLogger.Info("Worker stopped gracefully")
	case <-shutdownCtx.Done():
		appLogger.Warn("Worker shutdown timeout exceeded, forcing exit")
	}

	// Cleanup function to close all resources
	cleanup := func() {
		if dbClient != nil {
			dbClient.Close()
		}
		if rabbitClient != nil {
			rabbitClient.Close()
		}
	}
	cleanup()

	appLogger.Info("Worker service shutdown complete")
	return nil
}

// initLogger initializes and configures the application logger
func initLogger(cfg *config.LoggingConfig) (*logger.Logger, error) {
	loggerCfg := &logger.Config{
		Level:        cfg.Level,
		Format:       cfg.Format,
		Output:       cfg.Output,
		EnableSource: cfg.EnableCaller,
		TimeFormat:   time.RFC3339,
	}

	return logger.New(loggerCfg)
}

// initPostgreSQL initializes the PostgreSQL database client
func initPostgreSQL(cfg *config.DatabaseConfig, logger *slog.Logger) (*postgresql.Client, error) {
	dbConfig := &postgresql.Config{
		Host:            cfg.Host,
		Port:            cfg.Port,
		User:            cfg.User,
		Password:        cfg.Password,
		Database:        cfg.Database,
		SSLMode:         cfg.SSLMode,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	}

	return postgresql.NewClient(dbConfig, logger)
}

// initRabbitMQ initializes the RabbitMQ client
func initRabbitMQ(cfg *config.RabbitMQConfig, logger *slog.Logger) (*rabbitmq.Client, error) {
	rabbitConfig := &rabbitmq.Config{
		Host:               cfg.Host,
		Port:               cfg.Port,
		User:               cfg.User,
		Password:           cfg.Password,
		VHost:              cfg.VHost,
		ExchangeName:       cfg.Exchange.Name,
		ExchangeType:       cfg.Exchange.Type,
		ExchangeDurable:    cfg.Exchange.Durable,
		ExchangeAutoDelete: cfg.Exchange.AutoDelete,
		QueueName:          cfg.Queue.Name,
		QueueDurable:       cfg.Queue.Durable,
		QueueAutoDelete:    cfg.Queue.AutoDelete,
		QueueExclusive:     cfg.Queue.Exclusive,
		RoutingKey:         cfg.RoutingKey,
		RetryAttempts:      cfg.Connection.RetryAttempts,
		RetryInterval:      cfg.Connection.RetryInterval,
		Heartbeat:          cfg.Connection.Heartbeat,
		ConnectionTimeout:  cfg.Connection.ConnectionTimeout,
	}

	return rabbitmq.NewClient(rabbitConfig, logger)
}
