package game

import (
	"encoding/json"
	"log"

	"annonymity/models"
)

func SendToPlayer(p *models.Player, msgType string, payload interface{}) {
	if p == nil || p.Send == nil {
		return
	}
	data, err := buildMessage(msgType, payload)
	if err != nil {
		log.Printf("Error building message %s: %v", msgType, err)
		return
	}
	select {
	case p.Send <- data:
	default:
		log.Printf("Send channel full for player %s, dropping message %s", p.ID, msgType)
	}
}

func BroadcastToRoom(room *models.Room, msgType string, payload interface{}) {
	data, err := buildMessage(msgType, payload)
	if err != nil {
		log.Printf("Error building message %s: %v", msgType, err)
		return
	}
	for _, p := range room.Players {
		if p.Status == models.PlayerConnected && p.Send != nil {
			select {
			case p.Send <- data:
			default:
				log.Printf("Send channel full for player %s, dropping message %s", p.ID, msgType)
			}
		}
	}
}

func BroadcastToRoomExcept(room *models.Room, exceptID string, msgType string, payload interface{}) {
	data, err := buildMessage(msgType, payload)
	if err != nil {
		log.Printf("Error building message %s: %v", msgType, err)
		return
	}
	for _, p := range room.Players {
		if p.ID != exceptID && p.Status == models.PlayerConnected && p.Send != nil {
			select {
			case p.Send <- data:
			default:
			}
		}
	}
}

func SendError(p *models.Player, code, message string) {
	SendToPlayer(p, "error", models.ErrorPayload{
		Code:    code,
		Message: message,
	})
}

func buildMessage(msgType string, payload interface{}) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msg := struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}{
		Type:    msgType,
		Payload: payloadBytes,
	}
	return json.Marshal(msg)
}
