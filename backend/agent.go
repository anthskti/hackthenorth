package main

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	errWrongMode   = errors.New("only available in AI mode")
	errNotIdle     = errors.New("agent is not idle")
	errNotPausable = errors.New("agent is not running")
	errNotPaused   = errors.New("agent is not paused")
	errEmptyGoal   = errors.New("goal is required")
)

const defaultAgentMaxSteps = 8

type runPhase int

const (
	phaseNone     runPhase = 0
	phaseThinking runPhase = 1
	phaseActing   runPhase = 2
)

type AgentLoop struct {
	mu        sync.Mutex
	store     *Store
	hub       *Hub
	llm       *LLMClient
	pi        *PiClient
	frames    *FrameBuffer
	maxSteps  int
	goal      string
	paused    bool
	phase     runPhase
	cancel    context.CancelFunc
}

func agentMaxSteps() int {
	v := strings.TrimSpace(os.Getenv("AGENT_MAX_STEPS"))
	if v == "" {
		return defaultAgentMaxSteps
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultAgentMaxSteps
	}
	if n > 30 {
		return 30
	}
	return n
}

func NewAgentLoop(store *Store, hub *Hub, llm *LLMClient, pi *PiClient, frames *FrameBuffer) *AgentLoop {
	return &AgentLoop{
		store:    store,
		hub:      hub,
		llm:      llm,
		pi:       pi,
		frames:   frames,
		maxSteps: agentMaxSteps(),
	}
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
	a.maxSteps = agentMaxSteps()
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

	a.hub.BroadcastLog("Goal: " + goal)
	var history []AgentDecision
	goalComplete := false

	for step := 1; step <= a.maxSteps; step++ {
		if ctx.Err() != nil {
			return
		}
		if !a.waitUnpaused(ctx) {
			return
		}

		a.setPhase(phaseThinking)
		a.store.SetAgentState(StateThinking)
		a.hub.BroadcastLog("Step " + strconv.Itoa(step) + "/" + strconv.Itoa(a.maxSteps) + " — snapshot, " + a.llm.model + "…")

		frame := a.captureFrame(ctx)
		decision, rawJSON, err := a.llm.Decide(ctx, goal, frame, history)
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

		if decision != nil {
			history = append(history, *decision)
		}

		if decision != nil && (decision.Done || decision.Action == "done") {
			a.hub.BroadcastLog("Agent marked goal complete")
			goalComplete = true
			break
		}

		sentHID := a.dispatch(ctx, decision)
		if ctx.Err() != nil {
			return
		}

		if decision != nil && decision.Action == "wait" {
			_ = a.wait(ctx, 1500*time.Millisecond)
			continue
		}

		if sentHID {
			_ = a.wait(ctx, 1000*time.Millisecond)
		}
	}

	if !goalComplete {
		a.hub.BroadcastLog("Stopped — max steps (" + strconv.Itoa(a.maxSteps) + ") reached")
	}

	a.mu.Lock()
	a.goal = ""
	a.mu.Unlock()
	a.store.SetAgentState(StateIdle)
	a.hub.BroadcastLog("Idle")
}

func (a *AgentLoop) captureFrame(ctx context.Context) []byte {
	if a.pi != nil && a.pi.Enabled() {
		snap, err := a.pi.Snapshot(ctx)
		if err == nil && len(snap) > 0 {
			a.frames.Put(snap)
			a.store.SetPiConnected(true)
			return snap
		}
		if err != nil {
			a.hub.BroadcastLog("QNX snapshot failed: " + err.Error())
			a.store.SetPiConnected(false)
		}
	}
	return a.frames.LatestOrMock()
}

func (a *AgentLoop) dispatch(ctx context.Context, d *AgentDecision) bool {
	if d == nil {
		return false
	}
	if d.Action == "wait" || d.Action == "done" {
		return false
	}
	if a.pi == nil || !a.pi.Enabled() {
		a.hub.BroadcastLog("No QNX_BASE_URL — would run " + d.Action + " " + d.Value)
		return false
	}
	if err := a.pi.SendDecision(ctx, d); err != nil {
		a.hub.BroadcastLog("QNX /key error: " + err.Error())
		return false
	}
	a.hub.BroadcastLog("Sent to kvmd: " + d.Action + " " + d.Value)
	return true
}

func (a *AgentLoop) wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
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
