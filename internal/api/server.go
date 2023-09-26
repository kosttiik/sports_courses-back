package api

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func StartServer() {
	log.Println("Server is starting up...")

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.LoadHTMLGlob("templates/*")

	r.GET("/home", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Main Website",
		})
	})

	r.Static("/image", "./resources")

	r.Run()

	log.Println("Server shutdown.")
}
