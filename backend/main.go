package main

import (
	"net/http"

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
	hub = NewHub(store)
	frames := NewFrameBuffer()
	llm := NewLLMClient()
	agent := NewAgentLoop(store, hub, llm, frames)

	if err := llm.Ready(); err != nil {
		store.AppendLog("OpenAI disabled: set OPENAI_API_KEY in backend/.env")
	} else {
		store.AppendLog("OpenAI ready (" + llm.model + ")")
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	registerAPI(router, store, hub, agent)

	router.GET("/video/stream", handleVideoStream(frames))
	router.GET("/ws/dashboard", hub.HandleDashboardWS)

	router.Run()
}
