package service

import (
	"math/rand"
	"sort"

	"github.com/mrbelka12000/team_distributor/internal/models"
)

func DistributeMembers(members []models.Member, teams int) []models.Team {
	sort.Slice(members, func(i, j int) bool { return members[i].Rating > members[j].Rating })

	team1 := members[:len(members)/2]
	team2 := members[len(members)/2:]
	pos := rand.Intn(2)
	team1[pos], team2[len(team2)-1] = team2[len(team2)-1], team1[pos]
	pos = rand.Intn(2)
	team1[len(team1)-pos-1], team2[pos] = team2[pos], team1[len(team1)-pos-1]

	return []models.Team{
		{Members: team1, Total: getTotal(team1)},
		{Members: team2, Total: getTotal(team2)},
	}
}

func getTotal(members []models.Member) (total int) {
	for _, m := range members {
		total += m.Rating
	}
	return
}
