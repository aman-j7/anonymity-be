package models

import "encoding/json"

// WSMessage is the envelope for all incoming WebSocket messages.
type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// --- Client → Server Payloads ---

type SubmitAnswerPayload struct {
	Text string `json:"text"`
}

type SubmitVotePayload struct {
	VotedForPlayerID string `json:"voted_for_player_id"`
}

type UpdateSettingsPayload struct {
	MaxPlayers    *int `json:"max_players,omitempty"`
	NumRounds     *int `json:"num_rounds,omitempty"`
	AnswerTimeSec *int `json:"answer_time_sec,omitempty"`
	VoteTimeSec   *int `json:"vote_time_sec,omitempty"`
}

type KickPlayerPayload struct {
	PlayerID string `json:"player_id"`
}

type EmojiReactPayload struct {
	Emoji string `json:"emoji"`
}

// --- Server → Client Payloads ---

type PlayerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Score  int    `json:"score"`
	IsHost bool   `json:"is_host"`
	Status string `json:"status"`
}

type ConnectedPayload struct {
	PlayerID  string           `json:"player_id"`
	RoomCode  string           `json:"room_code"`
	RoomState RoomStatePayload `json:"room_state"`
}

type RoomStatePayload struct {
	Status       string       `json:"status"`
	HostID       string       `json:"host_id"`
	Players      []PlayerInfo `json:"players"`
	Settings     RoomSettings `json:"settings"`
	CurrentRound *RoundInfo   `json:"current_round"`
}

type RoundInfo struct {
	RoundNumber   int    `json:"round_number"`
	TotalRounds   int    `json:"total_rounds"`
	Question      string `json:"question"`
	Phase         string `json:"phase"`
	TimeRemaining int    `json:"time_remaining_sec"`
	AnswersCount  int    `json:"answers_count"`
	VotesCount    int    `json:"votes_count"`
	TotalPlayers  int    `json:"total_players"`
}

type PlayerJoinedPayload struct {
	Player      PlayerInfo `json:"player"`
	PlayerCount int        `json:"player_count"`
}

type PlayerLeftPayload struct {
	PlayerID    string `json:"player_id"`
	PlayerName  string `json:"player_name"`
	Reason      string `json:"reason"`
	PlayerCount int    `json:"player_count"`
}

type SettingsUpdatedPayload struct {
	Settings RoomSettings `json:"settings"`
}

type GameStartedPayload struct {
	TotalRounds int          `json:"total_rounds"`
	Players     []PlayerInfo `json:"players"`
}

type NewRoundPayload struct {
	RoundNumber    int        `json:"round_number"`
	TotalRounds    int        `json:"total_rounds"`
	Question       string     `json:"question"`
	FeaturedPlayer PlayerInfo `json:"featured_player"`
	Deadline       string     `json:"deadline"`
	TimeLimitSec   int        `json:"time_limit_sec"`
}

type AnswerSubmittedPayload struct {
	AnswersCount int  `json:"answers_count"`
	TotalPlayers int  `json:"total_players"`
	AllAnswered  bool `json:"all_answered"`
}

type VotingStartPayload struct {
	RoundNumber  int            `json:"round_number"`
	Question     string         `json:"question"`
	Answers      []VotingAnswer `json:"answers"`
	Deadline     string         `json:"deadline"`
	TimeLimitSec int            `json:"time_limit_sec"`
}

type VotingAnswer struct {
	PlayerID string `json:"player_id"`
	Text     string `json:"text"`
}

type VoteSubmittedPayload struct {
	VotesCount   int  `json:"votes_count"`
	TotalPlayers int  `json:"total_players"`
	AllVoted     bool `json:"all_voted"`
}

type RoundResultsPayload struct {
	RoundNumber    int            `json:"round_number"`
	Question       string         `json:"question"`
	Results        []AnswerResult `json:"results"`
	Scores         []PlayerScore  `json:"scores"`
	NextRoundInSec int            `json:"next_round_in_sec"`
}

type AnswerResult struct {
	PlayerID   string   `json:"player_id"`
	PlayerName string   `json:"player_name"`
	AnswerText string   `json:"answer_text"`
	VoteCount  int      `json:"vote_count"`
	IsWinner   bool     `json:"is_winner"`
	Voters     []string `json:"voters"`
}

type PlayerScore struct {
	PlayerID    string `json:"player_id"`
	Name        string `json:"name"`
	RoundPoints int    `json:"round_points"`
	TotalScore  int    `json:"total_score"`
}

type GameOverPayload struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
	MVP         MVPInfo            `json:"mvp"`
}

type LeaderboardEntry struct {
	Rank       int    `json:"rank"`
	PlayerID   string `json:"player_id"`
	Name       string `json:"name"`
	TotalScore int    `json:"total_score"`
}

type MVPInfo struct {
	PlayerID   string `json:"player_id"`
	Name       string `json:"name"`
	TotalScore int    `json:"total_score"`
	RoundsWon  int    `json:"rounds_won"`
}

type KickedPayload struct {
	Reason string `json:"reason"`
}

type EmojiReactionPayload struct {
	PlayerName string `json:"player_name"`
	Emoji      string `json:"emoji"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PongPayload struct {
	ServerTime string `json:"server_time"`
}
