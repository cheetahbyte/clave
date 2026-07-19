package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxArtifactBytes = int64(64 << 20)
	defaultMemoryBudget     = int64(1280 << 20)
	bsdiffMemoryMultiplier  = int64(17)
)

type Config struct {
	APIURL            string
	WorkerToken       string
	RabbitMQURL       string
	MaxArtifactBytes  int64
	MemoryBudgetBytes int64
	EffectiveMaxBytes int64
	HTTPTimeout       time.Duration
}

func loadConfig() (Config, error) {
	maxArtifact, err := envPositiveInt64("DELTA_MAX_ARTIFACT_BYTES", defaultMaxArtifactBytes)
	if err != nil {
		return Config{}, err
	}
	memoryBudget, err := envPositiveInt64("DELTA_MEMORY_BUDGET_BYTES", defaultMemoryBudget)
	if err != nil {
		return Config{}, err
	}
	timeoutSeconds, err := envPositiveInt64("DELTA_HTTP_TIMEOUT_SECONDS", 600)
	if err != nil {
		return Config{}, err
	}
	config := Config{
		APIURL:           strings.TrimRight(envOr("API_URL", "http://localhost:8000"), "/"),
		WorkerToken:      strings.TrimSpace(os.Getenv("WORKER_TOKEN")),
		RabbitMQURL:      envOr("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		MaxArtifactBytes: maxArtifact, MemoryBudgetBytes: memoryBudget,
		HTTPTimeout: time.Duration(timeoutSeconds) * time.Second,
	}
	if config.WorkerToken == "" {
		return Config{}, fmt.Errorf("WORKER_TOKEN is required")
	}
	config.EffectiveMaxBytes = min(maxArtifact, memoryBudget/bsdiffMemoryMultiplier)
	if config.EffectiveMaxBytes <= 0 {
		return Config{}, fmt.Errorf("DELTA_MEMORY_BUDGET_BYTES is too small for BSDIFF")
	}
	return config, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envPositiveInt64(key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return value, nil
}
