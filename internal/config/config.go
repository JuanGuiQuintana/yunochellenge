package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration grouped by concern.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	App      AppConfig
}

// ServerConfig holds HTTP server tuning parameters.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds pgxpool connection parameters.
type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Env           string
	LogLevel      string
	RunMigrations bool
}

// Load reads configuration from environment variables and returns a Config.
// It returns an error if any required variable is missing or any value is invalid.
func Load() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required but not set")
	}

	appEnv := envString("APP_ENV", "development")

	readTimeout, err := envDuration("SERVER_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("SERVER_READ_TIMEOUT: %w", err)
	}

	writeTimeout, err := envDuration("SERVER_WRITE_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("SERVER_WRITE_TIMEOUT: %w", err)
	}

	idleTimeout, err := envDuration("SERVER_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("SERVER_IDLE_TIMEOUT: %w", err)
	}

	maxConns, err := envInt32("DB_MAX_CONNS", 25)
	if err != nil {
		return Config{}, fmt.Errorf("DB_MAX_CONNS: %w", err)
	}

	minConns, err := envInt32("DB_MIN_CONNS", 5)
	if err != nil {
		return Config{}, fmt.Errorf("DB_MIN_CONNS: %w", err)
	}

	maxConnLifetime, err := envDuration("DB_MAX_CONN_LIFETIME", time.Hour)
	if err != nil {
		return Config{}, fmt.Errorf("DB_MAX_CONN_LIFETIME: %w", err)
	}

	maxConnIdleTime, err := envDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute)
	if err != nil {
		return Config{}, fmt.Errorf("DB_MAX_CONN_IDLE_TIME: %w", err)
	}

	runMigrations := appEnv != "production"
	if val := os.Getenv("RUN_MIGRATIONS"); val != "" {
		runMigrations, err = strconv.ParseBool(val)
		if err != nil {
			return Config{}, fmt.Errorf("RUN_MIGRATIONS: %w", err)
		}
	}

	return Config{
		Server: ServerConfig{
			Port:         envString("PORT", "8080"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		Database: DatabaseConfig{
			URL:             dbURL,
			MaxConns:        maxConns,
			MinConns:        minConns,
			MaxConnLifetime: maxConnLifetime,
			MaxConnIdleTime: maxConnIdleTime,
		},
		App: AppConfig{
			Env:           appEnv,
			LogLevel:      envString("LOG_LEVEL", "info"),
			RunMigrations: runMigrations,
		},
	}, nil
}

func envString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", val, err)
	}
	return d, nil
}

func envInt32(key string, defaultVal int32) (int32, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	n, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", val, err)
	}
	return int32(n), nil
}
