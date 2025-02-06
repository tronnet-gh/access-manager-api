package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "0.0.1",
		})
	})

	router.GET("/users", func(c *gin.Context) {})
	router.POST("/users/:userid", func(c *gin.Context) {
		var user User
		if err := c.ShouldBind(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user.Id = c.Param("userid")
		c.JSON(http.StatusOK, user)
	})
	router.GET("/users/:userid", func(c *gin.Context) {})
	router.DELETE("/users/:userid", func(c *gin.Context) {})

	router.GET("/groups", func(c *gin.Context) {})
	router.POST("/groups/:groupid", func(c *gin.Context) {})
	router.GET("/groups/:groupid", func(c *gin.Context) {})
	router.DELETE("/groups/:groupid", func(c *gin.Context) {})

	router.POST("/groups/:groupid/members/:userid", func(c *gin.Context) {})
	router.DELETE("/groups/:groupid/members/:userid", func(c *gin.Context) {})

	router.Run("0.0.0.0:80") // listen and serve on 0.0.0.0:8080
}
