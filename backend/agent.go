package main

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	errWrongMode    = errors.New("only available in AI mode")
	errNotIdle      = errors.New("agent is not idle")
	errNotPausable  = errors.New("agent is not running")
	errNotPaused    = errors.New("agent is not paused")
	errEmptyGoal    = errors.New("goal is required")
)

type runPhase int

const (
	phaseNone     runPhase = 0
	phaseThinking runPhase = 1
	phaseActing   runPhase = 2
)

type AgentLoop struct {
	mu     sync.Mutex
	store  *Store
	hub    *Hub
	llm    *LLMClient
	frames *FrameBuffer
	goal   string
	paused bool
	phase  runPhase
	cancel context.CancelFunc
}

func NewAgentLoop(store *Store, hub *Hub, llm *LLMClient, frames *FrameBuffer) *AgentLoop {
	return &AgentLoop{store: store, hub: hub, llm: llm, frames: frames}
}

func (a *AgentLoop) Stop() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.goal = ""
	a.paused = false
	a.mu.Unlock()
	a.store.SetAgentState(StateIdle)
}

func (a *AgentLoop) Start(goal string) error {
	goal = trimSpace(goal)
	if goal == "" {
		return errEmptyGoal
	}
	if a.store.Mode() != ModeAI {
		return errWrongMode
	}
	if a.store.AgentState() != StateIdle {
		return errNotIdle
	}
	if err := a.llm.Ready(); err != nil {
		return err
	}

	a.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.goal = goal
	a.paused = false
	a.cancel = cancel
	a.mu.Unlock()

	go a.run(ctx, goal)
	return nil
}

func (a *AgentLoop) Pause() error {
	if a.store.Mode() != ModeAI {
		return errWrongMode
	}
	st := a.store.AgentState()
	if st != StateThinking && st != StateActing {
		return errNotPausable
	}
	a.mu.Lock()
	a.paused = true
	a.mu.Unlock()
	a.store.SetAgentState(StatePaused)
	a.hub.BroadcastLog("Paused")
	return nil
}

func (a *AgentLoop) Resume() error {
	if a.store.Mode() != ModeAI {
		return errWrongMode
	}
	if a.store.AgentState() != StatePaused {
		return errNotPaused
	}
	a.mu.Lock()
	a.paused = false
	ph := a.phase
	a.mu.Unlock()
	switch ph {
	case phaseThinking:
		a.store.SetAgentState(StateThinking)
	case phaseActing:
		a.store.SetAgentState(StateActing)
	default:
		a.store.SetAgentState(StateIdle)
	}
	a.hub.BroadcastLog("Resumed")
	return nil
}

func (a *AgentLoop) StopUser() error {
	if a.store.Mode() != ModeAI {
		return errWrongMode
	}
	st := a.store.AgentState()
	if st != StateThinking && st != StateActing && st != StatePaused {
		return errNotPausable
	}
	a.Stop()
	a.hub.BroadcastLog("Stopped — goal cleared")
	return nil
}

func (a *AgentLoop) run(ctx context.Context, goal string) {
	defer func() {
		a.mu.Lock()
		a.cancel = nil
		a.phase = phaseNone
		a.mu.Unlock()
	}()

	a.setPhase(phaseThinking)
	a.store.SetAgentState(StateThinking)
	a.hub.BroadcastLog("Goal: " + goal)
	a.hub.BroadcastLog("Capturing frame, calling " + a.llm.model + "…")

	if !a.waitUnpaused(ctx) {
		return
	}

	frame := a.frames.LatestOrMock()
	decision, rawJSON, err := a.llm.Decide(ctx, goal, frame)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		a.hub.BroadcastLog("OpenAI error: " + err.Error())
		if len(rawJSON) > 0 {
			a.hub.BroadcastLog(string(rawJSON))
		}
		a.mu.Lock()
		a.goal = ""
		a.mu.Unlock()
		a.store.SetAgentState(StateIdle)
		return
	}

	if !a.waitUnpaused(ctx) {
		return
	}

	a.setPhase(phaseActing)
	a.store.SetAgentState(StateActing)
	a.hub.BroadcastLog(string(rawJSON))
	if decision != nil && decision.Thought != "" {
		a.hub.BroadcastLog("Thought: " + decision.Thought)
	}

	a.mu.Lock()
	a.goal = ""
	a.mu.Unlock()
	a.store.SetAgentState(StateIdle)
	a.hub.BroadcastLog("Idle — JSON only, no HID dispatch yet")
}

func (a *AgentLoop) setPhase(p runPhase) {
	a.mu.Lock()
	a.phase = p
	a.mu.Unlock()
}

func (a *AgentLoop) waitUnpaused(ctx context.Context) bool {
	for {
		if ctx.Err() != nil {
			return false
		}
		a.mu.Lock()
		p := a.paused
		a.mu.Unlock()
		if !p {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}
