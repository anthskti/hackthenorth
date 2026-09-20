package main

import (
	"context"
	"errors"
	"fmt"
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

const defaultAgentMaxSteps = 16

type runPhase int

const (
	phaseNone     runPhase = 0
	phaseThinking runPhase = 1
	phaseActing   runPhase = 2
)

type AgentLoop struct {
	mu       sync.Mutex
	store    *Store
	hub      *Hub
	llm      *LLMClient
	pi       *PiClient
	frames   *FrameBuffer
	maxSteps int
	goal     string
	paused   bool
	phase    runPhase
	cancel   context.CancelFunc
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
	a.hub.BroadcastLog("Planning for ThinkPad T14s Gen 6 BIOS…")
	plan, planErr := a.llm.Plan(ctx, goal)
	if ctx.Err() != nil {
		return
	}
	if planErr != nil {
		a.hub.BroadcastLog("Plan failed (continuing without): " + planErr.Error())
		plan = nil
	} else if plan != nil {
		if plan.Issue != "" {
			a.hub.BroadcastLog("Issue: " + plan.Issue)
		}
		if plan.MenuPath != "" {
			a.hub.BroadcastLog("Menu: " + plan.MenuPath)
		}
		if plan.Verify != "" {
			a.hub.BroadcastLog("Verify: " + plan.Verify)
		}
		a.hub.BroadcastLog("Plan: " + plan.Summary)
		for i, s := range plan.Steps {
			a.hub.BroadcastLog(fmt.Sprintf("  %d. %s", i+1, s))
		}
	}

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
		origN := len(frame)
		frame = compressForLLM(frame)
		if origN > 0 {
			a.hub.BroadcastLog(fmt.Sprintf("LLM still %d → %d bytes", origN, len(frame)))
		}
		decision, rawJSON, err := a.llm.Decide(ctx, goal, frame, history, plan)
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

		if decision != nil && (decision.Done || decision.Action == "done") {
			history = append(history, *decision)
			a.hub.BroadcastLog("Agent marked goal complete")
			goalComplete = true
			break
		}

		navCommit := navbarArrow(decision)
		sentHID := a.dispatch(ctx, decision)
		if decision != nil {
			history = append(history, *decision)
		}
		if ctx.Err() != nil {
			return
		}

		if decision != nil && decision.Action == "wait" {
			_ = a.wait(ctx, 1500*time.Millisecond)
			continue
		}

		if sentHID {
			wait := 1000 * time.Millisecond
			if navCommit {
				wait = 1400 * time.Millisecond
			}
			_ = a.wait(ctx, wait)
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

	if navbarArrow(d) {
		if !a.wait(ctx, 220*time.Millisecond) {
			return true
		}
		if err := a.pi.SendKey(ctx, "ENTER"); err != nil {
			a.hub.BroadcastLog("QNX /key error (navbar ENTER): " + err.Error())
			return true
		}
		a.hub.BroadcastLog("Navbar commit: ENTER (open hovered tab)")
		d.Value = strings.TrimSpace(d.Value) + "+ENTER"
	}
	return true
}

func navbarArrow(d *AgentDecision) bool {
	if d == nil || d.Action != "key" {
		return false
	}
	k := strings.ToUpper(strings.TrimSpace(d.Value))
	if k != "UP" && k != "DOWN" {
		return false
	}
	pane := strings.ToLower(strings.TrimSpace(d.Pane))
	if pane == "right_content" || pane == "right" {
		return false
	}
	if pane == "left_nav" || pane == "left" || pane == "navbar" {
		return true
	}
	obs := strings.ToLower(d.Observation)
	if strings.Contains(obs, "right content") || strings.Contains(obs, "right pane") {
		return false
	}
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
