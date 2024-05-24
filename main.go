package main

import "github.com/gin-gonic/gin"

func main() {
	router := gin.Default()
	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "0.0.1",
		})
	})
	router.Run("0.0.0.0:80") // listen and serve on 0.0.0.0:8080
}
