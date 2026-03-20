package models

import "github.com/gorilla/websocket"

type PlayerStatus string

const (
	PlayerConnected    PlayerStatus = "connected"
	PlayerDisconnected PlayerStatus = "disconnected"
)

type Player struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Score   int             `json:"score"`
	Status  PlayerStatus    `json:"status"`
	IsHost  bool            `json:"is_host"`
	Conn    *websocket.Conn `json:"-"`
	Send    chan []byte      `json:"-"`
	ConnGen uint64          `json:"-"`
}
