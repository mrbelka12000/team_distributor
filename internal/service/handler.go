package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/mrbelka12000/team_distributor/internal/models"
)

const (
	distributeTimeout  = 30 * time.Second
	selectGameCustomID = "select_game"
	usageMessage       = "Tag me with a list of players to distribute, e.g.:\n" +
		"```\n@bot 2 teams\nАрман — 1872\nБека — 900\nМара — 1750\nЖандос — 974\n```"
	sessionExpiredMsg = "Session expired — tag me again with the roster."
)

var mentionRe = regexp.MustCompile(`<@[!&]?\d+>`)

type Distributor interface {
	Distribute(ctx context.Context, rawRequest string, game models.Game) ([]models.Team, error)
}

type Handler struct {
	distributor Distributor
	sessions    *SessionStore
}

func NewHandler(d Distributor, sessions *SessionStore) *Handler {
	return &Handler{distributor: d, sessions: sessions}
}

func (h *Handler) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}
	if !isMentioned(m, s.State.User.ID) {
		return
	}

	log := slog.With(
		"channel_id", m.ChannelID,
		"guild_id", m.GuildID,
		"message_id", m.ID,
		"author_id", m.Author.ID,
		"author", m.Author.Username,
	)
	log.Info("mention received", "content_chars", len(m.Content))

	raw := strings.TrimSpace(stripMentions(m.Content))
	if raw == "" {
		log.Info("empty message after strip, sending usage")
		h.reply(s, m, usageMessage)
		return
	}

	members := ParseMembers(raw)
	log.Info("roster parsed", "members", len(members))
	if len(members) == 0 {
		log.Info("no members parsed, sending usage")
		h.reply(s, m, usageMessage)
		return
	}

	sent, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content:    fmt.Sprintf("Found %d players. Which game is this for?", len(members)),
		Reference:  m.Reference(),
		Components: gameSelectComponents(),
	})
	if err != nil {
		log.Error("send select menu failed", "error", err)
		return
	}

	h.sessions.Put(sent.ID, Session{
		Members: members,
		Origin:  m.Reference(),
	})
	log.Info("select menu sent", "reply_id", sent.ID, "members", len(members))
}

func (h *Handler) OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	data := i.MessageComponentData()
	if data.CustomID != selectGameCustomID || len(data.Values) == 0 {
		return
	}

	log := slog.With(
		"custom_id", data.CustomID,
		"message_id", i.Message.ID,
		"channel_id", i.ChannelID,
		"game_id", data.Values[0],
	)
	if user := interactionUser(i); user != nil {
		log = log.With("author_id", user.ID, "author", user.Username)
	}
	log.Info("interaction received")

	game, ok := models.GameByID(data.Values[0])
	if !ok {
		log.Warn("unknown game id")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		log.Error("ack interaction failed", "error", err)
		return
	}

	session, ok := h.sessions.Take(i.Message.ID)
	if !ok {
		log.Warn("session missing or expired")
		h.editInteraction(s, i.Interaction, sessionExpiredMsg)
		return
	}

	teams := DistributeMembers(session.Members, 2)
	content := fmt.Sprintf("**%s**\n%s", game.Name, formatTeams(teams))

	if _, err := s.ChannelMessageSendReply(i.ChannelID, content, session.Origin); err != nil {
		log.Error("send teams reply failed", "error", err)
		h.editInteraction(s, i.Interaction, content)
		return
	}

	if err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID); err != nil {
		log.Warn("delete prompt message failed", "error", err)
	}
}

func (h *Handler) reply(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	if _, err := s.ChannelMessageSendReply(m.ChannelID, content, m.Reference()); err != nil {
		slog.Error("send reply failed", "error", err, "channel_id", m.ChannelID, "message_id", m.ID)
	}
}

func (h *Handler) editInteraction(s *discordgo.Session, interaction *discordgo.Interaction, content string) {
	empty := []discordgo.MessageComponent{}
	if _, err := s.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Components: &empty,
	}); err != nil {
		slog.Error("edit interaction response failed", "error", err)
	}
}

func gameSelectComponents() []discordgo.MessageComponent {
	options := make([]discordgo.SelectMenuOption, 0, len(models.GameCatalog))
	for _, g := range models.GameCatalog {
		options = append(options, discordgo.SelectMenuOption{
			Label: g.Name,
			Value: g.ID,
		})
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    selectGameCustomID,
					Placeholder: "Pick a game",
					Options:     options,
				},
			},
		},
	}
}

func isMentioned(m *discordgo.MessageCreate, botID string) bool {
	for _, u := range m.Mentions {
		if u.ID == botID {
			return true
		}
	}
	return false
}

func interactionUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

func stripMentions(text string) string {
	return mentionRe.ReplaceAllString(text, "")
}

func formatMembers(members []models.Member) string {
	var sb strings.Builder
	for _, m := range members {
		fmt.Fprintf(&sb, "%s — %d\n", m.Name, m.Rating)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatTeams(teams []models.Team) string {
	var sb strings.Builder
	for i, t := range teams {
		name := t.Name
		if name == "" {
			name = fmt.Sprintf("Team %d", i+1)
		}
		fmt.Fprintf(&sb, "**%s** — total: %d\n", name, t.Total)
		for _, member := range t.Members {
			fmt.Fprintf(&sb, "• %s — %d\n", member.Name, member.Rating)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
