package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()

	hub := newHub()
	go hub.run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		sendOpenChannels(hub, w, r)
	})
	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		serveWebSocket(hub, w, r)
	})
	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		subscribeToClient(hub, w, r)
	})

	server := &http.Server{
		Addr:              *addr,
		ReadHeaderTimeout: 3 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func sendOpenChannels(hub *Hub, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var channels []string
	for ip := range hub.clients {
		channels = append(channels, ip)
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(channels)
	if err != nil {
		log.Println(err)
	}
}

func subscribeToClient(hub *Hub, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subscriptionIP := r.URL.Query().Get("ip")
	if subscriptionIP == "" {
		http.Error(w, "pass ip", http.StatusMethodNotAllowed)
		return
	}

	websocketIP, err := r.Cookie("ws-ip")
	if err != nil {
		log.Println(err)
		return
	}

	client, exists := hub.clients[websocketIP.Value]
	if !exists {
		http.Error(w, "client not found", http.StatusMethodNotAllowed)
		return
	}

	subscribingTo, exists := hub.clients[subscriptionIP]
	if !exists {
		http.Error(w, "subscription client not found", http.StatusMethodNotAllowed)
		return
	}

	subscribingTo.subscribe(client)
}

func serveWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	ip := readUserIP(r)
	client := newClient(hub, ip)
	client.ListenAndServe(w, r)
}

func readUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}
