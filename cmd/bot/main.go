package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrbelka12000/team_distributor/config"
	discordclient "github.com/mrbelka12000/team_distributor/internal/clients/discord"
	openaiclient "github.com/mrbelka12000/team_distributor/internal/clients/openai"
	"github.com/mrbelka12000/team_distributor/internal/service"
)

const (
	openaiPingTimeout = 10 * time.Second
	sessionTTL        = 10 * time.Minute
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info("starting team_distributor bot")

	cfg, err := config.Load(ctx)
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "openai_model", cfg.OpenAIModel)

	session, err := discordclient.New(cfg.DiscordToken)
	if err != nil {
		slog.Error("create discord session failed", "error", err)
		os.Exit(1)
	}
	slog.Info("discord session created")

	ai := openaiclient.New(cfg.OpenAIAPIKey, cfg.OpenAIModel)

	pingCtx, pingCancel := context.WithTimeout(ctx, openaiPingTimeout)
	err = ai.Ping(pingCtx)
	pingCancel()
	if err != nil {
		slog.Error("openai pre-check failed", "error", err, "model", ai.Model())
		os.Exit(1)
	}
	slog.Info("openai ready", "model", ai.Model())

	sessions := service.NewSessionStore(sessionTTL)
	slog.Info("session store initialised", "ttl", sessionTTL)

	handler := service.NewHandler(ai, sessions)
	session.AddHandler(handler.OnMessageCreate)
	session.AddHandler(handler.OnInteractionCreate)
	slog.Debug("discord handlers registered")

	if err := session.Open(); err != nil {
		slog.Error("open discord session failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := session.Close(); err != nil {
			slog.Error("close discord session", "error", err)
		}
	}()

	slog.Info("bot running", "username", session.State.User.Username, "id", session.State.User.ID)

	<-ctx.Done()
	slog.Info("shutdown signal received, stopping")
}
