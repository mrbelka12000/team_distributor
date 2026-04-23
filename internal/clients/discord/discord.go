package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

func New(token string) (*discordgo.Session, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("new discord session: %w", err)
	}

	s.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildMembers

	slog.Debug("discord session configured", "intents", s.Identify.Intents)
	return s, nil
}
