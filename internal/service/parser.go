package service

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mrbelka12000/team_distributor/internal/models"
)

var (
	listPrefixRe = regexp.MustCompile(`^\s*(?:[-*•·▪●►▫◦]+|\d+[.)])\s*`)
	dashPairRe   = regexp.MustCompile(`^(.+?)\s*[—–\-:]\s*(\d+)\s*$`)
	spacePairRe  = regexp.MustCompile(`^(.+?)\s+(\d+)\s*$`)
)

// ParseMembers extracts players from a freeform message. It tolerates list
// prefixes (bullets, numbered items) and common separators between name and
// rating (em dash, en dash, hyphen, colon, or whitespace).
func ParseMembers(raw string) []models.Member {
	var out []models.Member
	for _, rawLine := range strings.Split(raw, "\n") {
		line := listPrefixRe.ReplaceAllString(rawLine, "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if member, ok := extractMember(line); ok {
			out = append(out, member)
		}
	}
	return out
}

func extractMember(line string) (models.Member, bool) {
	for _, re := range []*regexp.Regexp{dashPairRe, spacePairRe} {
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(m[1])
		rating, err := strconv.Atoi(m[2])
		if err != nil || name == "" || rating <= 0 {
			continue
		}
		return models.Member{Name: name, Rating: rating}, true
	}
	return models.Member{}, false
}
