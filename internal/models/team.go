package models

type Member struct {
	Name   string `json:"name"`
	Rating int    `json:"rating"`
}

type Team struct {
	Name    string   `json:"name"`
	Total   int      `json:"total"`
	Members []Member `json:"members"`
}
