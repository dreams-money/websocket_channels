package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // must be less than pong wait
	maxMessageSize = 512
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	ip   string

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

func (c *Client) ListenAndServe(w http.ResponseWriter, r *http.Request) {
	var err error
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	c.conn, err = upgrader.Upgrade(w, r, http.Header{"Set-Cookie": []string{"ws-ip=" + c.ip}})
	if err != nil {
		log.Println(err)
		return
	}
	c.hub.register <- c

	go c.listenForIncomingMessage()
	go c.serveMessages()
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
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
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

func (c *Client) serveMessages() {
	heartbeat := time.NewTicker(pingPeriod)
	defer func() {
		heartbeat.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.sendClose()
				return
			}
			c.broadcastMessageToSubscribers(string(message))
		case <-heartbeat.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) broadcastMessageToSubscribers(message string) {
	for ip, subscriber := range c.subscribers {
		err := subscriber.sendMessage("broadcast", message)
		if err != nil {
			log.Println("failed to send message to", ip, err)
			continue
		}
	}
}

func (c *Client) sendMessage(messageType, messageBody string) error {
	message := message{
		Type: messageType,
		Body: messageBody,
	}
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))

	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonMessage)
	if err != nil {
		return err
	}

	return w.Close()
}

func (c *Client) sendClose() {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}
