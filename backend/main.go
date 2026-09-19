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

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	router.GET("/api/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, store.Snapshot())
	})

	router.GET("/video/stream", handleVideoStream)
	router.GET("/ws/dashboard", hub.HandleDashboardWS)

	router.Run()
}
