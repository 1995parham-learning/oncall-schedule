package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds the application configuration.
type Config struct {
	Server ServerConfig `koanf:"server"`
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Address string `koanf:"address"`
	Port    int    `koanf:"port"`
}

// Load loads configuration from file and environment variables.
func Load() (*Config, error) {
	k := koanf.New(".")

	// Load default configuration
	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		// Config file is optional, log but don't fail
		fmt.Printf("warning: config file not found: %v\n", err)
	}

	// Load environment variables with ONCALL_ prefix
	// Environment variables like ONCALL_SERVER_PORT will override config.yaml
	if err := k.Load(env.Provider("ONCALL_", ".", func(s string) string {
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set defaults if not provided
	if cfg.Server.Address == "" {
		cfg.Server.Address = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 1373
	}

	return &cfg, nil
}
