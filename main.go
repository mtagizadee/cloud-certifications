package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func setDB(database *gorm.DB) {
	db = database
}

func getDB(database *gorm.DB) *gorm.DB {
	return db
}

func main() {
	r := gin.Default()
	api := r.Group("/api")
	v1 := api.Group("/v1")

	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/authdb?charset=utf8mb4&parseTime=True&loc=Local"))
	if err != nil {
		panic("Could not connect to the database.")
	}
	setDB(db)
	fmt.Println("Connected to the database.")

	err = db.AutoMigrate(&company{}, &app{})
	if err != nil {
		panic("Could not migrate the database.")
	}
	fmt.Println("Database migrated.")
	
	companies := v1.Group("/companies")
	companies.POST("/", addCompany)
	companies.GET("/:id", getCompanyById)
	companies.POST("/:id/apps", addApp);

	r.Run("localhost:3000");
}

/**
* binding - request body validation
* json - json response formatter
* gorm - model validation
*/

type company struct { 
	gorm.Model
	Name string `gorm:"not null" binding:"required,min=2"`
	Email string `gorm:"unique;not null" binding:"required,email"`
	Password string `gorm:"not null" binding:"required,min=4,max=16"`
	Apps []app `gorm:"foreignkey:CompanyID"`
}

func (c *company) hashPassword() {
	hash := sha256.Sum256([]byte(c.Password))
	c.Password = fmt.Sprintf("%x", hash)
}

func addCompany(c *gin.Context) {
		var company company;
		if err := c.ShouldBindJSON(&company); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		company.hashPassword()

		db := getDB(db)
		err := db.Create(&company).Error
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, company)
}

func getCompanyById(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	db := getDB(db)
	var company company

	err := db.Model(&company).Preload("Apps").Where("id = ?", id).First(&company).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, company)
}

type app struct {
	gorm.Model
	Name string `gorm:"not null" binding:"required,min=2"`
	CompanyID int
}

func addApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	companyId, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var application app
	if err := c.ShouldBind(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	application.CompanyID = companyId

	err = db.Create(&application).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, application)
}