package internal

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
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
	JWTPrivateKey        string        `mapstructure:"jwt_private_key" validate:"required"`
	JWTPublicKey         string        `mapstructure:"jwt_public_key" validate:"required"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration" validate:"required,min=1m,max=1h"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration" validate:"required,min=1h"`
	BCryptCost           int           `mapstructure:"bcrypt_cost" validate:"required,min=10,max=15"`
	SessionSecret        string        `mapstructure:"session_secret" validate:"required,min=32"`
}

type PaymentConfig struct {
	MockAPIURL string `mapstructure:"mock_api_url" validate:"required,url"`
	APIKey     string `mapstructure:"api_key"`
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

// ----------------- HELPERS -----------------

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

// ----------------- VALIDATION -----------------

func (c *Config) Validate() error {
	var errs []string

	if err := c.Server.Validate(); err != nil {
		errs = append(errs, fmt.Sprintf("server config: %v", err))
	}

	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Sprintf("database config: %v", err))
	}

	if err := c.Security.Validate(); err != nil {
		errs = append(errs, fmt.Sprintf("security config: %v", err))
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

func (c *SecurityConfig) Validate() error {
	if _, err := c.GetPrivateKey(); err != nil {
		return fmt.Errorf("invalid JWT private key: %w", err)
	}
	if _, err := c.GetPublicKey(); err != nil {
		return fmt.Errorf("invalid JWT public key: %w", err)
	}
	if len(c.SessionSecret) < 32 {
		return errors.New("session secret must be at least 32 characters")
	}
	return nil
}

func (c *SecurityConfig) GetPrivateKey() (*rsa.PrivateKey, error) {
	keyData, err := base64.StdEncoding.DecodeString(c.JWTPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func (c *SecurityConfig) GetPublicKey() (*rsa.PublicKey, error) {
	keyData, err := base64.StdEncoding.DecodeString(c.JWTPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsaPub, nil
}

func (c *PaymentConfig) Validate() error {
	if c.MockAPIURL == "" {
		return errors.New("mock_api_url is required")
	}
	return nil
}
