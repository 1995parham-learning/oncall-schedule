package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const prefix = "ONCALL_"

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

	// load environment variables
	if err := k.Load(
		// replace __ with . in environment variables so you can reference field a in struct b
		// as a__b.
		env.Provider(".", env.Opt{
			Prefix: prefix,
			TransformFunc: func(source string, value string) (string, any) {
				base := strings.ToLower(strings.TrimPrefix(source, prefix))

				return strings.ReplaceAll(base, "__", "."), value
			},
			EnvironFunc: os.Environ,
		},
		),
		nil,
	); err != nil {
		fmt.Printf("error loading environment variables: %s", err)
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
