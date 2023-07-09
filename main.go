package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	r := gin.Default()
	api := r.Group("/api")
	v1 := api.Group("/v1")

	_, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/authdb?charset=utf8mb4&parseTime=True&loc=Local"))
	if err != nil {
		panic("Could not connect to the database.")
	}
	fmt.Println("Connected to the database.")
	
	

	v1.POST("/companies", addCompany)
	r.Run("localhost:3000");
}

func addCompany(c *gin.Context) {
	// temporary implementation
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}