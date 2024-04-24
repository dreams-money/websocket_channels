package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Client IP
	ip string

	// Subscribers: ip => client
	subscribers map[string]*Client
}

func newClient(hub *Hub, ip string) *Client {
	client := &Client{
		send:        make(chan []byte, maxMessageSize),
		subscribers: make(map[string]*Client),
		hub:         hub,
		ip:          ip,
	}
	// client.subscribe(client)
	return client
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (c *Client) ListenAndServe(w http.ResponseWriter, r *http.Request) {
	var err error
	c.conn, err = upgrader.Upgrade(w, r, http.Header{"Set-Cookie": []string{"ws-ip=" + c.ip}})
	if err != nil {
		log.Println(err)
		return
	}
	c.hub.register <- c

	go c.listenForIncomingMessage()
	go c.serveMessageToSubscribers()
}

func (c *Client) subscribe(subscriber *Client) {
	_, exists := c.subscribers[subscriber.ip]
	if exists {
		return
	}
	c.subscribers[subscriber.ip] = subscriber
}

func (c *Client) listenForIncomingMessage() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, []byte{'\n'}, []byte{' '}, -1))
		fmt.Println("message", string(message))
		c.send <- message
	}
}

func (c *Client) serveMessageToSubscribers() {
	heartbeat := time.NewTicker(pingPeriod)
	defer func() {
		heartbeat.Stop()
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
			for _, subscriber := range c.subscribers {
				subscriber.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					subscriber.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}

				w, err := subscriber.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					log.Println(err)
					return
				}
				w.Write(message)

				err = w.Close()
				if err != nil {
					log.Println(err)
					return
				}
			}

		case <-heartbeat.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		}
	}
}
