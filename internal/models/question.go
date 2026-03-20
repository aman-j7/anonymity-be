package models

import "strings"

type Question struct {
	ID       string `json:"id"`
	Template string `json:"template"`
	Category string `json:"category"`
}

func (q Question) Render(playerName string) string {
	return strings.ReplaceAll(q.Template, "{player}", playerName)
}
