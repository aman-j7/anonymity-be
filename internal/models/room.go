package models

import (
	"sync"
	"time"
)

type RoomStatus string

const (
	RoomStatusLobby    RoomStatus = "lobby"
	RoomStatusPlaying  RoomStatus = "playing"
	RoomStatusFinished RoomStatus = "finished"
)

type RoomSettings struct {
	MaxPlayers    int `json:"max_players"`
	NumRounds     int `json:"num_rounds"`
	AnswerTimeSec int `json:"answer_time_sec"`
	VoteTimeSec   int `json:"vote_time_sec"`
	ResultTimeSec int `json:"result_time_sec"`
}

func DefaultSettings() RoomSettings {
	return RoomSettings{
		MaxPlayers:    8,
		NumRounds:     5,
		AnswerTimeSec: 60,
		VoteTimeSec:   30,
		ResultTimeSec: 10,
	}
}

type Room struct {
	Mu            sync.RWMutex       `json:"-"`
	Code          string             `json:"code"`
	HostID        string             `json:"host_id"`
	Status        RoomStatus         `json:"status"`
	Players       map[string]*Player `json:"players"`
	Rounds        []*Round           `json:"rounds"`
	CurrentRound  int                `json:"current_round"`
	Settings      RoomSettings       `json:"settings"`

	Questions        []Question         `json:"-"`
	QuestionIdx      int                `json:"-"`
	UsedQuestionIDs  map[string]bool    `json:"-"` 

	FeaturedOrder []string      `json:"-"`
	FeaturedIdx   int           `json:"-"`
	CreatedAt     time.Time     `json:"created_at"`
	LastActivity  time.Time     `json:"-"`
	PhaseTimer    *time.Timer   `json:"-"`
}


func (r *Room) CurrentRoundData() *Round {
	if r.CurrentRound < 0 || r.CurrentRound >= len(r.Rounds) {
		return nil
	}
	return r.Rounds[r.CurrentRound]
}

func (r *Room) ActivePlayerCount() int {
	count := 0
	for _, p := range r.Players {
		if p.Status == PlayerConnected {
			count++
		}
	}
	return count
}
