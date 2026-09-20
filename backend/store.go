package main

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

const maxLogs = 200

type LogEntry struct {
	Text string `json:"text"`
	Ts   int64  `json:"ts"`
}

type Mode string

const (
	ModeAI     Mode = "ai"
	ModeManual Mode = "manual"
)

type AgentState string

const (
	StateIdle     AgentState = "idle"
	StateThinking AgentState = "thinking"
	StateActing   AgentState = "acting"
	StatePaused   AgentState = "paused"
)

type StatusResponse struct {
	Mode          Mode        `json:"mode"`
	State         *AgentState `json:"state,omitempty"`
	Logs          []LogEntry  `json:"logs"`
	PiConnected   bool        `json:"pi_connected"`
	StreamWidth   int         `json:"stream_width"`
	StreamHeight  int         `json:"stream_height"`
	QnxKeyUrl     string      `json:"qnx_key_url,omitempty"`
}

type Store struct {
	mu           sync.RWMutex
	mode         Mode
	state        AgentState
	logs         []LogEntry
	piConnected  bool
	streamWidth  int
	streamHeight int
	qnxKeyUrl    string
	onChange     func()
}

func NewStore(onChange func()) *Store {
	s := &Store{
		mode:         ModeAI,
		state:        StateIdle,
		piConnected:  false,
		streamWidth:  854,
		streamHeight: 480,
		onChange:     onChange,
	}
	s.appendLog("BOS backend ready")
	return s
}

func (s *Store) appendLog(text string) {
	entry := LogEntry{Text: text, Ts: time.Now().UnixMilli()}
	s.logs = append(s.logs, entry)
	if len(s.logs) > maxLogs {
		s.logs = s.logs[len(s.logs)-maxLogs:]
	}
}

func (s *Store) AppendLog(text string) {
	s.AppendLogEntry(text)
}

func (s *Store) AppendLogEntry(text string) LogEntry {
	s.mu.Lock()
	s.appendLog(text)
	entry := s.logs[len(s.logs)-1]
	s.mu.Unlock()
	return entry
}

func (s *Store) Snapshot() StatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resp := StatusResponse{
		Mode:         s.mode,
		Logs:         append([]LogEntry(nil), s.logs...),
		PiConnected:  s.piConnected,
		StreamWidth:  s.streamWidth,
		StreamHeight: s.streamHeight,
		QnxKeyUrl:    s.qnxKeyUrl,
	}
	if s.mode == ModeAI {
		st := s.state
		resp.State = &st
	}
	return resp
}

func (s *Store) Mode() Mode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *Store) AgentState() AgentState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Store) SetMode(m Mode) {
	s.mu.Lock()
	s.mode = m
	if m == ModeManual {
		s.state = StateIdle
	}
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Store) SetAgentState(st AgentState) {
	s.mu.Lock()
	s.state = st
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Store) StatusMessage() []byte {
	snap := s.Snapshot()
	payload := map[string]interface{}{
		"type": "status",
		"mode": snap.Mode,
	}
	if snap.State != nil {
		payload["state"] = *snap.State
	}
	b, _ := json.Marshal(payload)
	return b
}

func (s *Store) LogMessage(text string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"type": "log",
		"text": text,
		"ts":   time.Now().UnixMilli(),
	})
	return b
}

func (s *Store) SetQnxKeyUrl(url string) {
	s.mu.Lock()
	s.qnxKeyUrl = strings.TrimSpace(url)
	s.mu.Unlock()
}

func (s *Store) SetStreamSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	s.mu.Lock()
	s.streamWidth = width
	s.streamHeight = height
	s.mu.Unlock()
}

func (s *Store) SetPiConnected(connected bool) {
	s.mu.Lock()
	if s.piConnected == connected {
		s.mu.Unlock()
		return
	}
	s.piConnected = connected
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}
