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
			h.mu.Lock()
			h.clients[client.ip] = client
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.RLock()
			_, exists := h.clients[client.ip]
			h.mu.RUnlock()
			if exists {
				h.mu.Lock()
				delete(h.clients, client.ip)
				h.mu.Unlock()
				close(client.send)
			} else {
				log.Printf("unknown client: %v", client.ip)
			}
		}
	}
}
