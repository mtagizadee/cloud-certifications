package main

import (
	"fmt"

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

/**
* binding - request body validation
* json - json response formatter
* gorm - model validation
*/

type company struct { 
	gorm.Model
	Name string `gorm:"not null" json:"name" binding:"required,min=2"`
	Email string `gorm:"unique;not null" json:"email" binding:"required,email"`
	Password string `gorm:"not null" json:"-" binding:"required,min=4,max=16"`
}

func addCompany(c *gin.Context) {
	
}