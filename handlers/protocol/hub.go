package protocol

import (
	"encoding/json"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
	"log"
)

type Hub struct {
	app        core.App
	msgStore   map[string][]socketMessage
	clients    map[string]*Client
	broadcast  chan socketMessage
	register   chan *Client
	unregister chan *Client
}

func NewHub(app core.App) *Hub {
	return &Hub{
		app:        app,
		broadcast:  make(chan socketMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		msgStore:   make(map[string][]socketMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.ID()] = client
			messageRecord, err := h.app.Dao().FindFirstRecordByData("messages", "user_id", client.ID())
			if err != nil {
				log.Println("Error finding messages record", err)
				continue
			}

			pendingMessages, err := h.getPendingMessages(messageRecord)
			if err != nil {
				log.Println("Failed to fetch pending messages", err)
				continue
			}

			for _, message := range pendingMessages {
				h.clients[client.ID()].send <- message
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client.ID()]; ok {
				delete(h.clients, client.ID())
				close(client.send)
			}
		case message := <-h.broadcast:
			chatRecord, err := h.app.Dao().FindFirstRecordByData("chats", "id", message.ChatId)
			if err != nil {
				log.Println("error finding chat record", err)
				continue
			}

			chatParticipants := chatRecord.GetStringSlice("participants")
			for _, participant := range chatParticipants {
				if participant == message.Sender {
					continue
				}
				msgBytes, err := h.marshalSocketMessage(message, chatRecord)
				if err != nil {
					log.Println("Failed to serialize socket message", err)
					continue
				}

				if _, ok := h.clients[participant]; ok {
					h.clients[participant].send <- msgBytes
				} else {
					messageRecord, err := h.app.Dao().FindFirstRecordByData("messages", "user_id", participant)
					if err != nil {
						log.Println("Error finding messages record", err)
						continue
					}

					pendingMessages, err := h.getPendingMessages(messageRecord)
					if err != nil {
						log.Println("Failed to fetch pending messages", err)
						continue
					}

					pendingMessages = append(pendingMessages, msgBytes)
					messageRecord.Set("messages", pendingMessages)
					if err := h.app.Dao().SaveRecord(messageRecord); err != nil {
						log.Println("Failed to store pending message for offline user", err)
						continue
					}
				}
			}

		}
	}
}

func (h *Hub) marshalSocketMessage(message socketMessage, chatRecord *models.Record) ([]byte, error) {
	client := h.clients[message.Sender]
	chatType := chatRecord.GetString("type")
	if isDM(chatType) {
		message.Sender = client.chatUser.name
	} else {
		message.Sender = client.chatUser.tag
	}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	return msgBytes, nil
}

func (h *Hub) getPendingMessages(messageRecord *models.Record) ([][]byte, error) {
	var messages [][]byte
	pendingMessages := messageRecord.Get("messages").(types.JsonRaw)

	err := json.Unmarshal(pendingMessages, &messages)
	if err != nil {
		log.Println("Failed to deserialize pending messages", err)
		return nil, err
	}

	return messages, nil
}

func isDM(chatType string) bool {
	return chatType == "dm"
}
