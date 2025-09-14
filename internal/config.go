package internal

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server        ServerConfig        `mapstructure:"http_server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Security      SecurityConfig      `mapstructure:"security" validate:"required"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Payment       PaymentConfig       `mapstructure:"payment"`
}

type ServerConfig struct {
	Port              int           `mapstructure:"port"`
	BaseURL           string        `mapstructure:"base_url"`
	AllowedOrigins    string        `mapstructure:"allowed_origins"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	MaxOpenConns    int           `mapstructure:"max_open_conns" validate:"required,min=1"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" validate:"required,min=1"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" validate:"required,min=1m"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" validate:"required,min=1m"`
	Source          string        `mapstructure:"source"`
}

type SecurityConfig struct {
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration" validate:"required,min=1m,max=1h"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration" validate:"required,min=1h"`
	BCryptCost           int           `mapstructure:"bcrypt_cost" validate:"required,min=10,max=15"`
	SessionSecret        string        `mapstructure:"session_secret" validate:"required,min=32"`
}

type PaymentConfig struct {
	MockAPIURL     string        `mapstructure:"mock_api_url" validate:"required,url"`
	APIKey         string        `mapstructure:"api_key"`
	PaymentTimeout time.Duration `mapstructure:"payment_timeout" validate:"required,min=1s"`
	WebhookURL     string        `mapstructure:"webhook_url" validate:"omitempty,url"`
	MaxWorkers     int           `mapstructure:"max_workers" validate:"min=1,max=100"`
	JobQueueSize   int           `mapstructure:"job_queue_size" validate:"min=10,max=10000"`
	WorkerPoolSize int           `mapstructure:"worker_pool_size" validate:"min=1,max=100"`
}

type ObservabilityConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
	Logging LoggingConfig `mapstructure:"logging"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path" validate:"required_if=Enabled true"`
}

type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	ServiceName  string  `mapstructure:"service_name" validate:"required_if=Enabled true"`
	SamplingRate float64 `mapstructure:"sampling_rate" validate:"min=0,max=1"`
	JaegerURL    string  `mapstructure:"jaeger_url" validate:"required_if=Enabled true,url"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"required,oneof=json text"`
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultVal
}

func LoadConfigFromEnv() *Config {
	return &Config{
		Server: ServerConfig{
			Port:              getEnvAsInt("APP_PORT", 8080),
			BaseURL:           getEnv("APP_BASE_URL", "http://localhost:8080"),
			AllowedOrigins:    getEnv("CORS_ALLOWED_ORIGINS", "*"),
			ReadHeaderTimeout: getEnvAsDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			IdleTimeout:       getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			WriteTimeout:      getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 60*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 30*time.Minute),
			Source:          buildDSNFromEnv(),
		},
		Security: SecurityConfig{
			AccessTokenDuration:  getEnvAsDuration("JWT_EXPIRY", 24*time.Hour),
			RefreshTokenDuration: getEnvAsDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
			BCryptCost:           getEnvAsInt("BCRYPT_COST", 12),
			SessionSecret:        getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		},
		Payment: PaymentConfig{
			MockAPIURL:     getEnv("PAYMENT_MOCK_API_URL", "https://1620e98f-7759-431c-a2aa-f449d591150b.mock.pstmn.io"),
			APIKey:         getEnv("PAYMENT_API_KEY", "mock-postman-api-key"),
			WebhookURL:     getEnv("PAYMENT_WEBHOOK_URL", "http://localhost:8080/webhooks/payment/callback"),
			MaxWorkers:     getEnvAsInt("PAYMENT_MAX_WORKERS", 10),
			JobQueueSize:   getEnvAsInt("PAYMENT_JOB_QUEUE_SIZE", 100),
			WorkerPoolSize: getEnvAsInt("PAYMENT_WORKER_POOL_SIZE", 10),
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Level:  getEnv("LOG_LEVEL", "info"),
				Format: getEnv("LOG_FORMAT", "json"),
			},
			Metrics: MetricsConfig{
				Enabled: getEnv("METRICS_ENABLED", "false") == "true",
				Path:    getEnv("METRICS_PATH", "/metrics"),
			},
			Tracing: TracingConfig{
				Enabled:      getEnv("TRACING_ENABLED", "false") == "true",
				ServiceName:  getEnv("TRACING_SERVICE_NAME", "expense-management"),
				SamplingRate: 0.1,
				JaegerURL:    getEnv("JAEGER_URL", ""),
			},
		},
	}
}

func buildDSNFromEnv() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "expense_management")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func (c *Config) Validate() error {
	var errs []string

	if err := c.Server.Validate(); err != nil {
		errs = append(errs, fmt.Sprintf("server config: %v", err))
	}

	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Sprintf("database config: %v", err))
	}

	if err := c.Payment.Validate(); err != nil {
		errs = append(errs, fmt.Sprintf("payment config: %v", err))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func (c *ServerConfig) Validate() error {
	if c.AllowedOrigins != "" {
		origins := strings.Split(c.AllowedOrigins, ",")
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if origin == "*" {
				continue
			}
			if _, err := url.Parse(origin); err != nil {
				return fmt.Errorf("invalid allowed origin %s: %w", origin, err)
			}
		}
	}
	if c.ReadTimeout < c.ReadHeaderTimeout {
		return errors.New("read_timeout must be >= read_header_timeout")
	}
	return nil
}

func (c *DatabaseConfig) Validate() error {
	if c.MaxIdleConns > c.MaxOpenConns {
		return errors.New("max_idle_conns cannot be greater than max_open_conns")
	}
	return nil
}

func (c *DatabaseConfig) GetDSN() string {
	return c.Source
}

func (c *PaymentConfig) Validate() error {
	if c.MockAPIURL == "" {
		return errors.New("mock_api_url is required")
	}
	return nil
}
