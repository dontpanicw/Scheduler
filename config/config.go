package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"os"
	"strconv"
	"time"
)

const (
	DefaultMaxOpenConns      int32         = 25
	DefaultMaxLifetime       time.Duration = 5 * time.Minute
	DefaultMaxIdleTime       time.Duration = 1 * time.Minute
	DefaultSchedulerInterval time.Duration = 1 * time.Minute
	DefaultHTTPPort          string        = ":8090"
)

type Config struct {
	PG                PGConfig
	HTTPPort          string
	SchedulerInterval time.Duration
	NatsURL           string
}

type PGConfig struct {
	DSN          string        `yaml:"dsn"`
	MaxOpenConns int32         `yaml:"max_open_conns"`
	MaxLifetime  time.Duration `yaml:"max_lifetime"`
	MaxIdleTime  time.Duration `yaml:"max_idle_time"`
}

func NewConfig(logger *zap.Logger) (Config, error) {

	cfg := Config{}

	if err := godotenv.Load(); err != nil {
		logger.Warn("could not load .env file",
			zap.Error(err),
		)
	}

	pgDsn := os.Getenv("PG_DSN")
	if pgDsn == "" {
		logger.Info("PG_DSN not found, default value applied")
		return cfg, fmt.Errorf("PG_DSN environment variable is required\"")
	}
	cfg.PG.DSN = pgDsn

	maxOpenConnsStr := os.Getenv("PG_MAX_OPEN_CONNS")
	if maxOpenConnsStr == "" {
		cfg.PG.MaxOpenConns = DefaultMaxOpenConns
	} else {
		maxOpenConns, err := strconv.ParseInt(maxOpenConnsStr, 10, 32)
		if err != nil {
			return cfg, fmt.Errorf("invalid PG_MAX_OPEN_CONNS: %w", err)
		}
		cfg.PG.MaxOpenConns = int32(maxOpenConns)
	}

	maxLifetimeStr := os.Getenv("PG_MAX_LIFETIME")
	if maxLifetimeStr == "" {
		cfg.PG.MaxLifetime = DefaultMaxLifetime
	} else {
		maxLifetime, err := time.ParseDuration(maxLifetimeStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid PG_MAX_LIFETIME: %w", err)
		}
		cfg.PG.MaxLifetime = maxLifetime
	}

	// Parse MaxIdleTime
	maxIdleTimeStr := os.Getenv("PG_MAX_IDLE_TIME")
	if maxIdleTimeStr == "" {
		cfg.PG.MaxIdleTime = DefaultMaxIdleTime
	} else {
		maxIdleTime, err := time.ParseDuration(maxIdleTimeStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid PG_MAX_IDLE_TIME: %w", err)
		}
		cfg.PG.MaxIdleTime = maxIdleTime
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		return cfg, fmt.Errorf("NATS_URL environment variable is required")
	}
	cfg.NatsURL = natsURL

	rawInterval := os.Getenv("SCHEDULER_INTERVAL")
	if rawInterval == "" {
		cfg.SchedulerInterval = DefaultSchedulerInterval
	} else {
		interval, err := time.ParseDuration(rawInterval)
		if err != nil {
			return cfg, fmt.Errorf("invalid SCHEDULER_INTERVAL: %w", err)
		}
		cfg.SchedulerInterval = interval
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		cfg.HTTPPort = DefaultHTTPPort
	} else {
		// Ensure port starts with ':' if not already present
		if len(httpPort) > 0 && httpPort[0] != ':' {
			cfg.HTTPPort = ":" + httpPort
		} else {
			cfg.HTTPPort = httpPort
		}
	}
	return cfg, nil
}
