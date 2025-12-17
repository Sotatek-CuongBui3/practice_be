package postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Config holds PostgreSQL connection configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Client represents a PostgreSQL database client
type Client struct {
	db     *sqlx.DB
	config *Config
	logger *slog.Logger
}

// NewClient creates a new PostgreSQL client
func NewClient(config *Config, logger *slog.Logger) (*Client, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	logger.Info("Connecting to PostgreSQL",
		slog.String("host", config.Host),
		slog.Int("port", config.Port),
		slog.String("database", config.Database),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logger.Error("Failed to connect to PostgreSQL",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("Failed to ping PostgreSQL",
			slog.Any("error", err),
		)
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	client := &Client{
		db:     db,
		config: config,
		logger: logger,
	}

	logger.Info("Successfully connected to PostgreSQL",
		slog.Int("max_open_conns", config.MaxOpenConns),
		slog.Int("max_idle_conns", config.MaxIdleConns),
		slog.Duration("conn_max_lifetime", config.ConnMaxLifetime),
	)

	return client, nil
}

// GetDB returns the underlying sqlx.DB instance
func (c *Client) GetDB() *sqlx.DB {
	return c.db
}

// Close closes the database connection
func (c *Client) Close() error {
	c.logger.Info("Closing PostgreSQL connection")

	if c.db != nil {
		if err := c.db.Close(); err != nil {
			c.logger.Error("Failed to close PostgreSQL connection",
				slog.Any("error", err),
			)
			return err
		}
	}

	c.logger.Info("PostgreSQL connection closed successfully")
	return nil
}

// Ping checks the database connection
func (c *Client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// BeginTx starts a new transaction
func (c *Client) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		c.logger.Error("Failed to begin transaction",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// ExecContext executes a query without returning any rows
func (c *Client) ExecContext(ctx context.Context, query string, args ...interface{}) error {
	_, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		c.logger.Error("Failed to execute query",
			slog.Any("error", err),
			slog.String("query", query),
		)
		return fmt.Errorf("failed to execute query: %w", err)
	}
	return nil
}

// GetContext executes a query and scans a single row into dest
func (c *Client) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	err := c.db.GetContext(ctx, dest, query, args...)
	if err != nil {
		c.logger.Error("Failed to get row",
			slog.Any("error", err),
			slog.String("query", query),
		)
		return fmt.Errorf("failed to get row: %w", err)
	}
	return nil
}

// SelectContext executes a query and scans multiple rows into dest
func (c *Client) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	err := c.db.SelectContext(ctx, dest, query, args...)
	if err != nil {
		c.logger.Error("Failed to select rows",
			slog.Any("error", err),
			slog.String("query", query),
		)
		return fmt.Errorf("failed to select rows: %w", err)
	}
	return nil
}

// NamedExecContext executes a named query without returning any rows
func (c *Client) NamedExecContext(ctx context.Context, query string, arg interface{}) error {
	_, err := c.db.NamedExecContext(ctx, query, arg)
	if err != nil {
		c.logger.Error("Failed to execute named query",
			slog.Any("error", err),
			slog.String("query", query),
		)
		return fmt.Errorf("failed to execute named query: %w", err)
	}
	return nil
}

// NamedQueryContext executes a named query and returns rows
func (c *Client) NamedQueryContext(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	rows, err := c.db.NamedQueryContext(ctx, query, arg)
	if err != nil {
		c.logger.Error("Failed to execute named query",
			slog.Any("error", err),
			slog.String("query", query),
		)
		return nil, fmt.Errorf("failed to execute named query: %w", err)
	}
	return rows, nil
}

// Stats returns database statistics
func (c *Client) Stats() string {
	stats := c.db.Stats()
	return fmt.Sprintf(
		"MaxOpenConns: %d, OpenConns: %d, InUse: %d, Idle: %d, WaitCount: %d, WaitDuration: %s",
		stats.MaxOpenConnections,
		stats.OpenConnections,
		stats.InUse,
		stats.Idle,
		stats.WaitCount,
		stats.WaitDuration,
	)
}

// HealthCheck performs a health check on the database
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Try a simple query
	var result int
	err := c.db.GetContext(ctx, &result, "SELECT 1")
	if err != nil {
		return fmt.Errorf("database query health check failed: %w", err)
	}

	return nil
}
