package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize a default Gin router with Logger and Recovery middleware
	router := gin.Default()

	// Define a simple GET endpoint
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Run the server (listens on 0.0.0.0:8080 by default)
	router.Run() 
}
