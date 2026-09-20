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
	pi      *PiClient
}

func NewHub(store *Store, pi *PiClient) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		store:   store,
		pi:      pi,
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
	entry := h.store.AppendLogEntry(text)
	payload, _ := json.Marshal(map[string]interface{}{
		"type": "log",
		"text": entry.Text,
		"ts":   entry.Ts,
	})
	h.Broadcast(payload)
}

func (h *Hub) BroadcastStatus() {
	h.Broadcast(h.store.StatusMessage())
}

func (h *Hub) dropClient(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
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

	defer h.dropClient(conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
