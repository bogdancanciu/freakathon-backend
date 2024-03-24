package protocol

import (
	"encoding/json"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
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
			//if _, ok := h.msgStore[client.id]; ok {
			//	for _, msg := range h.msgStore[client.id] {
			//		msgBytes, err := json.Marshal(msg)
			//		if err != nil {
			//			log.Println("error while serializing message", err)
			//		}
			//		if _, ok := h.clients[msg.Receiver]; ok {
			//			h.clients[client.id].send <- msgBytes
			//		}
			//	}
			//	h.msgStore[client.id] = []socketMessage{}
			//}
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
				if _, ok := h.clients[participant]; ok {
					msgBytes, err := h.marshalSocketMessage(message, chatRecord)
					if err != nil {
						log.Println("Failed to serialize socket message", err)
						continue
					}

					h.clients[participant].send <- msgBytes
				} else {
					//h.msgStore[message.Receiver] = append(h.msgStore[message.Receiver], message)
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

func isDM(chatType string) bool {
	return chatType == "dm"
}
