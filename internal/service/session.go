package service

import (
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/mrbelka12000/team_distributor/internal/models"
)

type Session struct {
	Members []models.Member
	Origin  *discordgo.MessageReference
}

type sessionEntry struct {
	session   Session
	expiresAt time.Time
}

type SessionStore struct {
	mu      sync.Mutex
	entries map[string]sessionEntry
	ttl     time.Duration
}

func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{
		entries: make(map[string]sessionEntry),
		ttl:     ttl,
	}
}

func (s *SessionStore) Put(id string, session Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[id] = sessionEntry{
		session:   session,
		expiresAt: time.Now().Add(s.ttl),
	}
	purged := s.sweepLocked()
	slog.Debug("session put", "id", id, "members", len(session.Members), "live", len(s.entries), "purged", purged)
}

// Take returns the session for the given id and removes the entry. Returns
// false if missing or expired.
func (s *SessionStore) Take(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[id]
	if !ok {
		slog.Debug("session take miss", "id", id)
		return Session{}, false
	}
	delete(s.entries, id)
	if time.Now().After(entry.expiresAt) {
		slog.Debug("session take expired", "id", id)
		return Session{}, false
	}
	slog.Debug("session take hit", "id", id, "members", len(entry.session.Members), "live", len(s.entries))
	return entry.session, true
}

func (s *SessionStore) sweepLocked() int {
	now := time.Now()
	purged := 0
	for id, entry := range s.entries {
		if now.After(entry.expiresAt) {
			delete(s.entries, id)
			purged++
		}
	}
	return purged
}
