package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	loadDotEnv(".env")

	var hub *Hub
	store := NewStore(func() {
		if hub != nil {
			hub.BroadcastStatus()
		}
	})
	pi := NewPiClient()
	store.SetStreamSize(pi.streamWidth, pi.streamHeight)
	hub = NewHub(store, pi)
	frames := NewFrameBuffer()
	llm := NewLLMClient()
	agent := NewAgentLoop(store, hub, llm, pi, frames)

	if err := llm.Ready(); err != nil {
		store.AppendLog("OpenAI disabled: set OPENAI_API_KEY in backend/.env")
	} else {
		store.AppendLog("OpenAI ready (" + llm.model + ")")
	}

	if pi.Enabled() {
		store.AppendLog("QNX kvmd " + pi.stream)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			if err := pi.Probe(ctx); err != nil {
				store.SetPiConnected(false)
				store.AppendLog("QNX unreachable: " + err.Error())
				hub.BroadcastStatus()
				return
			}
			store.SetPiConnected(true)
			store.AppendLog("QNX snapshot OK")
			hub.BroadcastStatus()
		}()
	} else {
		store.AppendLog("QNX unset — mock video. Set QNX_BASE_URL in backend/.env")
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	registerAPI(router, store, hub, agent)

	router.GET("/video/stream", pi.HandleStream(frames, store))
	router.GET("/ws/dashboard", hub.HandleDashboardWS)

	router.Run()
}
