package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	customerror "anonymity/internal/error"
	"anonymity/internal/models"
	"anonymity/internal/rooms"
	"anonymity/internal/store"

	"github.com/go-chi/chi/v5"
)

type HTTPHandler struct {
	store     *store.GameStore
	startTime time.Time
}

func NewHTTPHandler(s *store.GameStore) *HTTPHandler {
	return &HTTPHandler{store: s, startTime: time.Now()}
}

type CreateRoomRequest struct {
	HostName string               `json:"host_name"`
	Settings *models.RoomSettings `json:"settings,omitempty"`
}

type CreateRoomResponse struct {
	RoomCode     string              `json:"room_code"`
	HostPlayerID string              `json:"host_player_id"`
	Settings     models.RoomSettings `json:"settings"`
}

type GetRoomResponse struct {
	RoomCode    string              `json:"room_code"`
	Status      string              `json:"status"`
	PlayerCount int                 `json:"player_count"`
	MaxPlayers  int                 `json:"max_players"`
	HostName    string              `json:"host_name"`
	Players     []RoomPlayerInfo    `json:"players"`
	Settings    models.RoomSettings `json:"settings"`
}

type RoomPlayerInfo struct {
	Name   string `json:"name"`
	IsHost bool   `json:"is_host"`
}

func (h *HTTPHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.HostName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host_name is required"})
		return
	}
	if len(req.HostName) > 30 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host_name must be 30 characters or less"})
		return
	}

	settings := models.DefaultSettings()
	if req.Settings != nil {
		if req.Settings.MaxPlayers > 0 {
			settings.MaxPlayers = req.Settings.MaxPlayers
		}
		if req.Settings.NumRounds > 0 {
			settings.NumRounds = req.Settings.NumRounds
		}
		if req.Settings.AnswerTimeSec > 0 {
			settings.AnswerTimeSec = req.Settings.AnswerTimeSec
		}
		if req.Settings.VoteTimeSec > 0 {
			settings.VoteTimeSec = req.Settings.VoteTimeSec
		}
		if req.Settings.ResultTimeSec > 0 {
			settings.ResultTimeSec = req.Settings.ResultTimeSec
		}
	}

	if settings.MaxPlayers < 3 || settings.MaxPlayers > 12 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "max_players must be between 3 and 12"})
		return
	}
	if settings.NumRounds < 1 || settings.NumRounds > 20 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "num_rounds must be between 1 and 20"})
		return
	}

	room, hostID, err := h.store.CreateRoom(req.HostName, settings, r.Context())
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, CreateRoomResponse{
		RoomCode:     room.Code,
		HostPlayerID: hostID,
		Settings:     room.Settings,
	})
}

func (h *HTTPHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room code is required"})
		return
	}

	room := h.store.GetRoom(code)
	if room == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}

	room.Mu.RLock()
	defer room.Mu.RUnlock()

	if room.Status != models.RoomStatusLobby {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "game already in progress"})
		return
	}
	if len(room.Players) >= room.Settings.MaxPlayers {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "room is full"})
		return
	}

	hostName := ""
	players := make([]RoomPlayerInfo, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, RoomPlayerInfo{
			Name:   p.Name,
			IsHost: p.IsHost,
		})
		if p.IsHost {
			hostName = p.Name
		}
	}

	writeJSON(w, http.StatusOK, GetRoomResponse{
		RoomCode:    room.Code,
		Status:      string(room.Status),
		PlayerCount: len(room.Players),
		MaxPlayers:  room.Settings.MaxPlayers,
		HostName:    hostName,
		Players:     players,
		Settings:    room.Settings,
	})
}

func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"active_rooms":   rooms.RoomCount(),
		"uptime_seconds": int(time.Since(h.startTime).Seconds()),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func handleError(w http.ResponseWriter, err any) {
	if appErr, ok := err.(*customerror.AppError); ok {
		writeJSON(w, appErr.Code, appErr)
		return
	}

	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"message": "Internal server error",
	})
}
