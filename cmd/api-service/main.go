package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cuongbtq/practice-be/internal/api/handler"
	"github.com/cuongbtq/practice-be/internal/api/router"
	"github.com/cuongbtq/practice-be/internal/config"
	"github.com/cuongbtq/practice-be/shared/logger"
	"github.com/cuongbtq/practice-be/shared/postgresql"
	"github.com/cuongbtq/practice-be/shared/rabbitmq"
	"github.com/gin-gonic/gin"
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
	defaultConfigPath := os.Getenv("API_SERVICE_CONFIG_PATH")
	if defaultConfigPath == "" {
		defaultConfigPath = "configs/api-service/config.yaml"
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

	appLogger.Info("Starting API service",
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

	// Initialize router
	r := initRouter(cfg.App.Environment, appLogger.Logger, dbClient, rabbitClient)

	// Create HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	appLogger.Info("Starting HTTP server",
		slog.String("address", addr),
		slog.Duration("read_timeout", cfg.Server.ReadTimeout),
		slog.Duration("write_timeout", cfg.Server.WriteTimeout),
	)

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed to start",
				slog.Any("error", err),
			)
			os.Exit(1)
		}
	}()

	appLogger.Info("API service is running",
		slog.String("address", addr),
	)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)

	// Cleanup function to close all resources
	cleanup := func() {
		cancel()
		if dbClient != nil {
			dbClient.Close()
		}
		if rabbitClient != nil {
			rabbitClient.Close()
		}
	}
	defer cleanup()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown",
			slog.Any("error", err),
		)
		return err
	}

	appLogger.Info("Server shutdown complete")
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
		PublishRetries:     cfg.Publish.RetryAttempts,
		PublishRetryDelay:  cfg.Publish.RetryInterval,
		PublishBackoffMult: cfg.Publish.BackoffMultiplier,
	}

	return rabbitmq.NewClient(rabbitConfig, logger)
}

// initRouter initializes the Gin router with all routes and middleware
func initRouter(environment string, logger *slog.Logger, dbClient *postgresql.Client, rabbitClient *rabbitmq.Client) *gin.Engine {
	// Set Gin mode based on environment
	if environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize handler dependencies
	handlerDeps := &handler.Dependencies{
		Logger:       logger,
		DBClient:     dbClient,
		RabbitClient: rabbitClient,
	}

	// Setup router
	return router.SetupRouter(handlerDeps)
}
