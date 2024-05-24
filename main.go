package main

import (
	"bytes"
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "0.0.1",
		})
	})

	router.POST("/echo", func(c *gin.Context) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(c.Request.Body)
		data := buf.String()
		fmt.Println(data)
	})
	router.Run("0.0.0.0:80") // listen and serve on 0.0.0.0:8080
}
