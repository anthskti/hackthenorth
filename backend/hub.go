package main

import (
	"context"
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
	mu          sync.Mutex
	clients     map[*websocket.Conn]struct{}
	manualMouse map[*websocket.Conn]*manualPointerState
	store       *Store
	pi          *PiClient
}

func NewHub(store *Store, pi *PiClient) *Hub {
	return &Hub{
		clients:     make(map[*websocket.Conn]struct{}),
		manualMouse: make(map[*websocket.Conn]*manualPointerState),
		store:       store,
		pi:          pi,
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

func (h *Hub) pointerState(conn *websocket.Conn) *manualPointerState {
	h.mu.Lock()
	st := h.manualMouse[conn]
	if st == nil {
		st = &manualPointerState{}
		h.manualMouse[conn] = st
	}
	h.mu.Unlock()
	return st
}

func (h *Hub) dropClient(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	delete(h.manualMouse, conn)
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
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		h.handleClientMessage(conn, data)
	}
}

func (h *Hub) handleClientMessage(conn *websocket.Conn, data []byte) {
	var msg struct {
		Type   string `json:"type"`
		Action string `json:"action"`
		Value  string `json:"value"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Button string `json:"button"`
		Delta  int    `json:"delta"`
	}
	if err := json.Unmarshal(data, &msg); err != nil || msg.Type != "manual_input" {
		return
	}
	if h.store.Mode() != ModeManual {
		return
	}
	if h.pi == nil || !h.pi.Enabled() {
		if msg.Action == "key" {
			h.BroadcastLog("No QNX_BASE_URL — key " + msg.Value)
		}
		return
	}

	ctx := context.Background()
	st := h.pointerState(conn)

	switch msg.Action {
	case "mouse_move":
		if err := h.pi.SendMouseMoveTo(ctx, st, msg.X, msg.Y); err != nil {
			h.BroadcastLog("QNX mouse move: " + err.Error())
		}
	case "mouse_click":
		btn := msg.Button
		if btn == "" {
			btn = "left"
		}
		if err := h.pi.SendMouseClick(ctx, st, msg.X, msg.Y, btn); err != nil {
			h.BroadcastLog("QNX mouse click: " + err.Error())
			return
		}
		h.BroadcastLog(fmt.Sprintf("Manual %s click at (%d, %d)", btn, msg.X, msg.Y))
	case "mouse_wheel":
		if err := h.pi.SendMouseWheel(ctx, msg.Delta); err != nil {
			h.BroadcastLog("QNX mouse wheel: " + err.Error())
		}
	case "key":
		if err := h.pi.SendKey(ctx, msg.Value); err != nil {
			h.BroadcastLog("QNX /key error: " + err.Error())
			return
		}
	}
}
