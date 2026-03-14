package models

import "time"

type RoundPhase string

const (
	PhaseAnswering RoundPhase = "answering"
	PhaseVoting    RoundPhase = "voting"
	PhaseResults   RoundPhase = "results"
)

type Answer struct {
	PlayerID    string    `json:"player_id"`
	PlayerName  string    `json:"player_name"`
	Text        string    `json:"text"`
	SubmittedAt time.Time `json:"submitted_at"`
	VoteCount   int       `json:"vote_count"`
	IsWinner    bool      `json:"is_winner"`
}

type Vote struct {
	VoterID        string    `json:"voter_id"`
	VotedForPlayer string    `json:"voted_for_player"`
	SubmittedAt    time.Time `json:"submitted_at"`
}

type Round struct {
	RoundNumber    int                `json:"round_number"`
	Question       Question           `json:"question"`
	RenderedPrompt string             `json:"rendered_prompt"`
	FeaturedPlayer string             `json:"featured_player"`
	Phase          RoundPhase         `json:"phase"`
	Answers        map[string]*Answer `json:"answers"`
	Votes          map[string]*Vote   `json:"votes"`
	StartedAt      time.Time          `json:"started_at"`
	PhaseDeadline  time.Time          `json:"phase_deadline"`
}
