package store

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"anonymity/models"
	"anonymity/utils"
)

type GameStore struct {
	mu    sync.RWMutex
	rooms map[string]*models.Room
}

func New() *GameStore {
	return &GameStore{
		rooms: make(map[string]*models.Room),
	}
}

func (s *GameStore) CreateRoom(hostName string, settings models.RoomSettings) (*models.Room, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code := utils.GenerateRoomCode(func(c string) bool {
		_, ok := s.rooms[c]
		return ok
	})

	hostID := uuid.New().String()

	host := &models.Player{
		ID:     hostID,
		Name:   hostName,
		Score:  0,
		Status: models.PlayerDisconnected,
		IsHost: true,
	}

	room := &models.Room{
		Code:         code,
		HostID:       hostID,
		Status:       models.RoomStatusLobby,
		Players:      map[string]*models.Player{hostID: host},
		Rounds:       make([]*models.Round, 0),
		CurrentRound: -1,
		Settings:     settings,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	s.rooms[code] = room
	return room, hostID
}

func (s *GameStore) GetRoom(code string) *models.Room {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rooms[code]
}

func (s *GameStore) DeleteRoom(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, code)
}

func (s *GameStore) RoomCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rooms)
}

func (s *GameStore) StartCleanup(interval, maxIdle time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanup(maxIdle)
		}
	}()
}

func (s *GameStore) cleanup(maxIdle time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for code, room := range s.rooms {
		room.Mu.Lock()
		shouldDelete := room.Status == models.RoomStatusFinished ||
			(room.ActivePlayerCount() == 0 && now.Sub(room.LastActivity) > maxIdle)

		if shouldDelete {
			if room.PhaseTimer != nil {
				room.PhaseTimer.Stop()
			}
			for _, p := range room.Players {
				if p.Send != nil {
					close(p.Send)
					p.Send = nil
				}
			}
			room.Mu.Unlock()
			delete(s.rooms, code)
			log.Printf("Cleaned up room %s", code)
		} else {
			room.Mu.Unlock()
		}
	}
}
