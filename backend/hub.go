package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
	store   *Store
}

func NewHub(store *Store) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		store:   store,
	}
}

func (h *Hub) Broadcast(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *Hub) BroadcastLog(text string) {
	h.store.AppendLog(text)
	h.Broadcast(h.store.LogMessage(text))
}

func (h *Hub) BroadcastStatus() {
	h.Broadcast(h.store.StatusMessage())
}

func (h *Hub) HandleDashboardWS(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	_ = conn.WriteMessage(websocket.TextMessage, h.store.StatusMessage())

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		h.handleClientMessage(data)
	}
}

func (h *Hub) handleClientMessage(data []byte) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || envelope.Type != "manual_input" {
		return
	}
	if h.store.Mode() != ModeManual {
		return
	}
	// Accepted in manual mode; Pi forward comes in a later step.
}
