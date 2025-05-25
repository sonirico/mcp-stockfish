package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Stockfish StockfishConfig
	Server    ServerConfig
	Logging   LoggingConfig
}

type StockfishConfig struct {
	Path           string
	MaxSessions    int
	SessionTimeout time.Duration
	CommandTimeout time.Duration
}

type ServerConfig struct {
	Name    string
	Version string
	Mode    string // "stdio" or "http"
	Host    string
	Port    int
	CORS    bool
}

type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

func loadConfig() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		Stockfish: StockfishConfig{
			Path:           getEnv("MCP_STOCKFISH_PATH", "stockfish"),
			MaxSessions:    getIntEnv("MCP_STOCKFISH_MAX_SESSIONS", 10),
			SessionTimeout: getDurationEnv("MCP_STOCKFISH_SESSION_TIMEOUT", 30*time.Minute),
			CommandTimeout: getDurationEnv("MCP_STOCKFISH_COMMAND_TIMEOUT", 30*time.Second),
		},
		Server: ServerConfig{
			Name:    getEnv("MCP_STOCKFISH_SERVER_NAME", "mcp-stockfish ♟️"),
			Version: version,
			Mode:    getEnv("MCP_STOCKFISH_SERVER_MODE", "stdio"),
			Host:    getEnv("MCP_STOCKFISH_HTTP_HOST", "localhost"),
			Port:    getIntEnv("MCP_STOCKFISH_HTTP_PORT", 8080),
			CORS:    getBoolEnv("MCP_STOCKFISH_HTTP_CORS", true),
		},
		Logging: LoggingConfig{
			Level:  getEnv("MCP_STOCKFISH_LOG_LEVEL", "info"),
			Format: getEnv("MCP_STOCKFISH_LOG_FORMAT", "console"),
			Output: getEnv("MCP_STOCKFISH_LOG_OUTPUT", "stderr"),
		},
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func validateConfig(config *Config) error {
	if config.Stockfish.MaxSessions <= 0 {
		return fmt.Errorf("max_sessions must be positive")
	}

	if config.Stockfish.SessionTimeout <= 0 {
		return fmt.Errorf("session_timeout must be positive")
	}

	if config.Server.Mode != "stdio" && config.Server.Mode != "http" {
		return fmt.Errorf("server mode must be 'stdio' or 'http'")
	}

	if config.Server.Mode == "http" {
		if config.Server.Port <= 0 || config.Server.Port > 65535 {
			return fmt.Errorf("invalid HTTP port: %d", config.Server.Port)
		}
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
