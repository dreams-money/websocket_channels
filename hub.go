package main

import (
	"log"
	"sync"
)

// Central hub that keeps track of clients
type Hub struct {
	// IP -> Client
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	h.clients[client.ip] = client
	h.mu.Unlock()
	h.sendToAllClients("connect", client.ip)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.RLock()
	_, exists := h.clients[client.ip]
	h.mu.RUnlock()
	if exists {
		h.mu.Lock()
		delete(h.clients, client.ip)
		h.mu.Unlock()
		close(client.send)
		h.sendToAllClients("disconnect", client.ip)
	} else {
		log.Printf("unknown client: %v", client.ip)
	}
}

func (h *Hub) sendToAllClients(messageType, messageBody string) {
	for ip, client := range h.clients {
		err := client.sendMessage(messageType, messageBody)
		if err != nil {
			log.Println("failed to send message to", ip, err)
		}
	}
}
