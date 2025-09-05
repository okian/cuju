package config

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Load builds a Config by layering defaults, optional file, and env vars.
// Order of precedence (low -> high):.
//  1. defaults (New(ctx))
//  2. file (YAML) if CUJU_CONFIG is set
//  3. env (prefix CUJU_)
func Load(ctx context.Context) (*Config, error) {
	// Start with defaults
	base := New()

	k := koanf.New(".")

	// Load from file if provided
	if path := os.Getenv("CUJU_CONFIG"); path != "" {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, err
		}
	}

	// Environment variables: CUJU_ADDR, CUJU_QUEUE_SIZE, ...
	// Map env keys like CUJU_QUEUE_SIZE -> queue_size (flat keys)
	// Preserve underscores to match koanf tags on the struct.
	envProvider := env.Provider("CUJU_", ".", func(s string) string {
		s = strings.ToLower(s)
		s = strings.TrimPrefix(s, "cuju_")
		return s
	})
	if err := k.Load(envProvider, nil); err != nil {
		return nil, err
	}

	// Unmarshal into a copy
	cfg := *base
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return nil, err
	}

	// Basic validation
	if cfg.Addr == "" {
		return nil, errors.New("addr must not be empty")
	}
	return &cfg, nil
}
