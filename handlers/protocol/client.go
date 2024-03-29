package protocol

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type socketMessage struct {
	ChatId    string `json:"chat_id"`
	Sender    string `json:"sender"`
	Content   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type chatUser struct {
	id   string
	name string
	tag  string
}

func newChatUser(id, name, tag string) *chatUser {
	return &chatUser{id: id, name: name, tag: tag}
}

type Client struct {
	chatUser *chatUser
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
}

func (c *Client) ID() string {
	return c.chatUser.id
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("failed to read socket message: %v", err)
			}
			break
		}

		var msg socketMessage
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Printf("failed to unmarshal socket message: %v", err)
			continue
		}

		msg.Sender = c.chatUser.id
		msg.Timestamp = time.Now().Unix()

		c.hub.broadcast <- msg
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
