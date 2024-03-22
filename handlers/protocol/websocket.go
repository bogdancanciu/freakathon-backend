package protocol

import (
	"errors"
	"github.com/bogdancanciu/frekathon-backend/handlers"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type sender struct {
	senderId string
	groupId  string
}

type storeMessage struct {
	Sender  sender `json:"sender"`
	Message string `json:"message"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from all origins
		return true
	},
}

func ServeWs(app core.App, hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	_, sessionToken, err := conn.ReadMessage()
	if err != nil {
		log.Println("Error reading initial message:", err)
		conn.Close()
		return
	}

	record, err := app.Dao().FindFirstRecordByData("messages", "user_id", "e4eymnms6hoyb69")
	if err != nil {
		log.Println("error while finding rec")
	}

	log.Println(record.Get("messages"))

	userId, err := handlers.UserIdFromSession(string(sessionToken))
	if !errors.Is(err, (*apis.ApiError)(nil)) {
		log.Println("Failed to decode session token", err)
		conn.Close()
		return
	}

	log.Println(userId)

	client := &Client{id: userId, hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
