package main

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerAPI(r *gin.Engine, store *Store, hub *Hub, agent *AgentLoop) {
	r.GET("/api/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, store.Snapshot())
	})

	r.POST("/api/mode", func(c *gin.Context) {
		var body struct {
			Mode Mode `json:"mode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if body.Mode != ModeAI && body.Mode != ModeManual {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be ai or manual"})
			return
		}

		prev := store.Mode()
		if body.Mode == ModeManual && prev == ModeAI {
			agent.Stop()
		}

		store.SetMode(body.Mode)
		if body.Mode == ModeManual {
			hub.BroadcastLog("Switched to manual mode")
		} else if prev == ModeManual {
			hub.BroadcastLog("Switched to AI mode")
		}
		hub.BroadcastStatus()
		c.JSON(http.StatusOK, store.Snapshot())
	})

	r.POST("/api/prompt", func(c *gin.Context) {
		var body struct {
			Goal string `json:"goal"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := agent.Start(body.Goal); err != nil {
			writeAgentError(c, err)
			return
		}
		hub.BroadcastStatus()
		c.JSON(http.StatusOK, store.Snapshot())
	})

	r.POST("/api/control", func(c *gin.Context) {
		var body struct {
			Action string `json:"action"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		var err error
		switch body.Action {
		case "pause":
			err = agent.Pause()
		case "resume":
			err = agent.Resume()
		case "stop":
			err = agent.StopUser()
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "action must be pause, resume, or stop"})
			return
		}
		if err != nil {
			writeAgentError(c, err)
			return
		}
		hub.BroadcastStatus()
		c.JSON(http.StatusOK, store.Snapshot())
	})
}

func writeAgentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errWrongMode):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, errNotIdle), errors.Is(err, errNotPausable), errors.Is(err, errNotPaused):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, errEmptyGoal), errors.Is(err, errNoAPIKey):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent error"})
	}
}
