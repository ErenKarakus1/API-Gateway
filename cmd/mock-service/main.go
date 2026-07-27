package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "mock-service"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9001"
	}

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"status":  "ok",
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"users": []gin.H{
				{"id": "1", "name": "Ada Lovelace"},
				{"id": "2", "name": "Grace Hopper"},
			},
		})
	})

	router.GET("/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"id":      c.Param("id"),
			"name":    "Portfolio User",
			"user_id": c.GetHeader("X-User-ID"),
		})
	})

	router.POST("/", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{
			"service": serviceName,
			"status":  "created",
			"user_id": c.GetHeader("X-User-ID"),
		})
	})

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
