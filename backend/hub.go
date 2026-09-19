package main

import (
	"encoding/json"
	"fmt"
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
	var msg struct {
		Type   string `json:"type"`
		Action string `json:"action"`
		Value  string `json:"value"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Button string `json:"button"`
	}
	if err := json.Unmarshal(data, &msg); err != nil || msg.Type != "manual_input" {
		return
	}
	if h.store.Mode() != ModeManual {
		return
	}
	// Pi / HID forward later; accept and acknowledge clicks only in the log.
	if msg.Action == "mouse_click" {
		h.BroadcastLog(fmt.Sprintf("Manual click at (%d, %d)", msg.X, msg.Y))
	}
}
