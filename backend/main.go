package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	var hub *Hub
	store := NewStore(func() {
		if hub != nil {
			hub.BroadcastStatus()
		}
	})
	hub = NewHub(store)
	agent := NewAgentLoop(store, hub)

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	registerAPI(router, store, hub, agent)

	router.GET("/video/stream", handleVideoStream)
	router.GET("/ws/dashboard", hub.HandleDashboardWS)

	router.Run()
}
