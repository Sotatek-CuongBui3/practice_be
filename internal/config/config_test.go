package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		wantErr   bool
		errString string
	}{
		{
			name:     "valid config file",
			filePath: "testdata/valid_config.yaml",
			wantErr:  false,
		},
		{
			name:      "non-existent file",
			filePath:  "testdata/nonexistent.yaml",
			wantErr:   true,
			errString: "failed to read config file",
		},
		{
			name:      "malformed yaml",
			filePath:  "testdata/malformed.yaml",
			wantErr:   true,
			errString: "failed to parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.filePath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)

				// Verify some key fields are populated
				assert.Equal(t, 8080, cfg.Server.Port)
				assert.Equal(t, "localhost", cfg.Database.Host)
				assert.Equal(t, 5432, cfg.Database.Port)
				assert.Equal(t, "jobs_db", cfg.Database.Database)
				assert.Equal(t, "jobs_exchange", cfg.RabbitMQ.Exchange.Name)
				assert.Equal(t, "jobs_queue", cfg.RabbitMQ.Queue.Name)
				assert.Equal(t, "job-api-service", cfg.App.Name)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantErr   bool
		errString string
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "jobs_db",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "jobs_exchange",
					},
					Queue: QueueConfig{
						Name: "jobs_queue",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid server port - too low",
			config: &Config{
				Server: ServerConfig{Port: 0},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "jobs_db",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "jobs_exchange",
					},
					Queue: QueueConfig{
						Name: "jobs_queue",
					},
				},
			},
			wantErr:   true,
			errString: "invalid server port",
		},
		{
			name: "invalid server port - too high",
			config: &Config{
				Server: ServerConfig{Port: 70000},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "jobs_db",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "jobs_exchange",
					},
					Queue: QueueConfig{
						Name: "jobs_queue",
					},
				},
			},
			wantErr:   true,
			errString: "invalid server port",
		},
		{
			name: "empty database host",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "",
					Port:     5432,
					Database: "jobs_db",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "jobs_exchange",
					},
					Queue: QueueConfig{
						Name: "jobs_queue",
					},
				},
			},
			wantErr:   true,
			errString: "database host is required",
		},
		{
			name: "empty database name",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "jobs_exchange",
					},
					Queue: QueueConfig{
						Name: "jobs_queue",
					},
				},
			},
			wantErr:   true,
			errString: "database name is required",
		},
		{
			name: "empty rabbitmq host",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "jobs_db",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "jobs_exchange",
					},
					Queue: QueueConfig{
						Name: "jobs_queue",
					},
				},
			},
			wantErr:   true,
			errString: "rabbitmq host is required",
		},
		{
			name: "empty exchange name",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "jobs_db",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "",
					},
					Queue: QueueConfig{
						Name: "jobs_queue",
					},
				},
			},
			wantErr:   true,
			errString: "rabbitmq exchange name is required",
		},
		{
			name: "empty queue name",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "jobs_db",
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
					Exchange: ExchangeConfig{
						Name: "jobs_exchange",
					},
					Queue: QueueConfig{
						Name: "",
					},
				},
			},
			wantErr:   true,
			errString: "rabbitmq queue name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoad_ValidateIntegration(t *testing.T) {
	t.Run("load and validate valid config", func(t *testing.T) {
		cfg, err := Load("testdata/valid_config.yaml")
		require.NoError(t, err)
		require.NotNil(t, cfg)

		err = cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("load config with invalid port", func(t *testing.T) {
		cfg, err := Load("testdata/invalid_port.yaml")
		require.NoError(t, err)
		require.NotNil(t, cfg)

		err = cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server port")
	})

	t.Run("load config with missing database", func(t *testing.T) {
		cfg, err := Load("testdata/missing_database.yaml")
		require.NoError(t, err)
		require.NotNil(t, cfg)

		err = cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database name is required")
	})
}

func TestPortConstants(t *testing.T) {
	t.Run("port constants are correct", func(t *testing.T) {
		assert.Equal(t, 1, MinPort)
		assert.Equal(t, 65535, MaxPort)
	})

	t.Run("valid port range", func(t *testing.T) {
		validPorts := []int{1, 80, 443, 8080, 65535}
		for _, port := range validPorts {
			assert.GreaterOrEqual(t, port, MinPort)
			assert.LessOrEqual(t, port, MaxPort)
		}
	})

	t.Run("invalid port range", func(t *testing.T) {
		invalidPorts := []int{0, -1, 65536, 70000}
		for _, port := range invalidPorts {
			valid := port >= MinPort && port <= MaxPort
			assert.False(t, valid, "port %d should be invalid", port)
		}
	})
}
