package models

type Game struct {
	ID       string
	Name     string
	TeamSize int
}

var GameCatalog = []Game{
	{ID: "dota2", Name: "Dota 2", TeamSize: 5},
	{ID: "cs2", Name: "CS2", TeamSize: 5},
	{ID: "left2dead2", Name: "L4D2", TeamSize: 4},
	{ID: "basketball", Name: "Basketball", TeamSize: 5},
	{ID: "custom", Name: "Custom", TeamSize: 0},
}

func GameByID(id string) (Game, bool) {
	for _, g := range GameCatalog {
		if g.ID == id {
			return g, true
		}
	}
	return Game{}, false
}
