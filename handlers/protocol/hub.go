package protocol

import (
	"encoding/json"
	"log"
)

type Hub struct {
	msgStore   map[string][]socketMessage
	clients    map[string]*Client
	broadcast  chan socketMessage
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
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
			h.clients[client.id] = client
			if _, ok := h.msgStore[client.id]; ok {
				for _, msg := range h.msgStore[client.id] {
					msgBytes, err := json.Marshal(msg)
					if err != nil {
						log.Println("error while serializing message", err)
					}
					if _, ok := h.clients[msg.Receiver]; ok {
						h.clients[client.id].send <- msgBytes
					}
				}
				h.msgStore[client.id] = []socketMessage{}
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client.id]; ok {
				delete(h.clients, client.id)
				close(client.send)
			}
		case message := <-h.broadcast:
			msgBytes, err := json.Marshal(message)
			if err != nil {
				log.Println("error while serializing message", err)
			}
			if _, ok := h.clients[message.Receiver]; ok {
				h.clients[message.Receiver].send <- msgBytes
			} else {
				h.msgStore[message.Receiver] = append(h.msgStore[message.Receiver], message)
			}
		}
	}
}
