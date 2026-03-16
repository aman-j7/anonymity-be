package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"anonymity/game"
	"anonymity/models"
	"anonymity/store"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	store  *store.GameStore
	engine *game.Engine
}

func NewWSHandler(s *store.GameStore, e *game.Engine) *WSHandler {
	return &WSHandler{store: s, engine: e}
}

func (h *WSHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	playerName := r.URL.Query().Get("name")
	playerID := r.URL.Query().Get("player_id")

	if roomCode == "" {
		http.Error(w, `{"error":"room parameter is required"}`, http.StatusBadRequest)
		return
	}

	room := h.store.GetRoom(roomCode)
	if room == nil {
		http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	room.Mu.Lock()

	var player *models.Player
	isReconnect := false

	if playerID != "" {
		player = room.Players[playerID]
		if player == nil {
			room.Mu.Unlock()
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4003, "invalid player_id"))
			conn.Close()
			return
		}

		oldSend := player.Send
		oldConn := player.Conn

		player.Conn = conn
		player.Send = make(chan []byte, 256)
		player.Status = models.PlayerConnected
		player.ConnGen++
		isReconnect = true

		if oldSend != nil {
			close(oldSend)
		}
		if oldConn != nil {
			oldConn.Close()
		}
	} else {
		if playerName == "" {
			room.Mu.Unlock()
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseProtocolError, "name parameter is required"))
			conn.Close()
			return
		}
		if room.Status != models.RoomStatusLobby {
			room.Mu.Unlock()
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4002, "game already in progress"))
			conn.Close()
			return
		}
		if len(room.Players) >= room.Settings.MaxPlayers {
			room.Mu.Unlock()
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4001, "room is full"))
			conn.Close()
			return
		}
		if len(playerName) > 30 {
			playerName = playerName[:30]
		}

		player = &models.Player{
			ID:      uuid.New().String(),
			Name:    playerName,
			Score:   0,
			Status:  models.PlayerConnected,
			IsHost:  false,
			Conn:    conn,
			Send:    make(chan []byte, 256),
			ConnGen: 1,
		}
		room.Players[player.ID] = player
	}

	room.LastActivity = time.Now()
	h.sendConnectedEvent(player, room)

	if !isReconnect {
		game.BroadcastToRoomExcept(room, player.ID, "player_joined", models.PlayerJoinedPayload{
			Player: models.PlayerInfo{
				ID:     player.ID,
				Name:   player.Name,
				Score:  player.Score,
				IsHost: player.IsHost,
				Status: string(player.Status),
			},
			PlayerCount: len(room.Players),
		})
	}

	localConn := player.Conn
	localSend := player.Send
	localGen := player.ConnGen

	room.Mu.Unlock()

	log.Printf("Player %s (%s) connected to room %s", player.Name, player.ID, room.Code)

	go h.writePump(localConn, localSend)
	h.readPump(localConn, localGen, player, room)
}

func (h *WSHandler) sendConnectedEvent(player *models.Player, room *models.Room) {
	players := make([]models.PlayerInfo, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, models.PlayerInfo{
			ID:     p.ID,
			Name:   p.Name,
			Score:  p.Score,
			IsHost: p.IsHost,
			Status: string(p.Status),
		})
	}

	var currentRound *models.RoundInfo
	if room.Status == models.RoomStatusPlaying {
		rd := room.CurrentRoundData()
		if rd != nil {
			remaining := int(time.Until(rd.PhaseDeadline).Seconds())
			if remaining < 0 {
				remaining = 0
			}
			currentRound = &models.RoundInfo{
				RoundNumber:   rd.RoundNumber,
				TotalRounds:   room.Settings.NumRounds,
				Question:      rd.RenderedPrompt,
				Phase:         string(rd.Phase),
				TimeRemaining: remaining,
				AnswersCount:  len(rd.Answers),
				VotesCount:    len(rd.Votes),
				TotalPlayers:  room.ActivePlayerCount(),
			}
		}
	}

	game.SendToPlayer(player, "connected", models.ConnectedPayload{
		PlayerID: player.ID,
		RoomCode: room.Code,
		RoomState: models.RoomStatePayload{
			Status:       string(room.Status),
			HostID:       room.HostID,
			Players:      players,
			Settings:     room.Settings,
			CurrentRound: currentRound,
		},
	})
}

func (h *WSHandler) writePump(conn *websocket.Conn, send <-chan []byte) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-send:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *WSHandler) readPump(conn *websocket.Conn, connGen uint64, player *models.Player, room *models.Room) {
	defer func() {
		h.engine.HandleDisconnect(player, room, connGen)
		conn.Close()
		log.Printf("Player %s (%s) disconnected from room %s", player.Name, player.ID, room.Code)
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error for %s: %v", player.Name, err)
			}
			break
		}

		var msg models.WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			game.SendError(player, "INVALID_PAYLOAD", "Malformed JSON")
			continue
		}

		h.routeMessage(player, room, msg)
	}
}

func (h *WSHandler) routeMessage(player *models.Player, room *models.Room, msg models.WSMessage) {
	switch msg.Type {
	case "start_game":
		h.engine.HandleStartGame(player, room)
	
	case "end_game":
		h.engine.HandleEndGame(player, room)

	case "submit_answer":
		var p models.SubmitAnswerPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			game.SendError(player, "INVALID_PAYLOAD", "Invalid answer payload")
			return
		}
		h.engine.HandleSubmitAnswer(player, room, p)

	case "submit_vote":
		var p models.SubmitVotePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			game.SendError(player, "INVALID_PAYLOAD", "Invalid vote payload")
			return
		}
		h.engine.HandleSubmitVote(player, room, p)

	case "update_settings":
		var p models.UpdateSettingsPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			game.SendError(player, "INVALID_PAYLOAD", "Invalid settings payload")
			return
		}
		h.engine.HandleUpdateSettings(player, room, p)

	case "kick_player":
		var p models.KickPlayerPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			game.SendError(player, "INVALID_PAYLOAD", "Invalid kick payload")
			return
		}
		h.engine.HandleKickPlayer(player, room, p)

	case "emoji_react":
		var p models.EmojiReactPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			game.SendError(player, "INVALID_PAYLOAD", "Invalid emoji payload")
			return
		}
		h.engine.HandleEmojiReact(player, room, p)

	case "ping":
		game.SendToPlayer(player, "pong", models.PongPayload{
			ServerTime: time.Now().Format(time.RFC3339),
		})

	default:
		game.SendError(player, "UNKNOWN_EVENT", "Unknown event type: "+msg.Type)
	}
}
