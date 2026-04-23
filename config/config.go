package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	DiscordToken string `env:"DISCORD_TOKEN, required"`
	OpenAIAPIKey string `env:"OPENAI_API_KEY, required"`
	OpenAIModel  string `env:"OPENAI_MODEL"`
}

func Load(ctx context.Context) (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Debug("no .env file loaded", "error", err)
	} else {
		slog.Debug(".env file loaded")
	}

	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("process env: %w", err)
	}
	return &cfg, nil
}
