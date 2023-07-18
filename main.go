package main

import (
	"auth/packages/_jwt"
	"auth/packages/db"
	"auth/packages/entities"
	"auth/packages/migration"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	api := r.Group("/api")
	v1 := api.Group("/v1")

	err := db.Connect()
	if err != nil {
		panic("Could not connect to the database.")
	}

	err = migration.Migrate()
	if err != nil {
		panic("Could not migrate the database.")
	}

	companies := v1.Group("/companies")
	companies.POST("/", addCompany)
	companies.GET("/:id", getCompanyById)
	companies.POST("/:id/apps", addApp);
	companies.POST(":id/apps/:appId/certificates", addCertificate)

	certificates := v1.Group("/certificates")
	certificates.POST("/verify", verifyCertificate)

	auth := v1.Group("/auth")
	auth.POST("/login", login)
		
	r.Run("localhost:3000");
}

func addCompany(c *gin.Context) {
		var company entities.Company;
		if err := c.ShouldBindJSON(&company); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		company.HashPassword()

		_db := db.GetDB()
		err := _db.Create(&company).Error
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

	_db := db.GetDB()
	var company entities.Company

	err := _db.Model(&company).Preload("Apps").Where("id = ?", id).First(&company).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, company)
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

	var application entities.App
	if err := c.ShouldBind(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	application.CompanyID = companyId

	_db := db.GetDB()
	err = _db.Create(&application).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, application)
}

func addCertificate(c *gin.Context) {
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

	strAppId := c.Param("appId")
	if strAppId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appId is required"})
		return
	}
	appId, err := strconv.Atoi(strAppId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// setup the certificate
	builder, err := entities.
										NewCertificate().
										SetAppId(appId).
										GenerateId().
										GenerateAccessToken(companyId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	err = builder.Prepare()	
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusCreated, builder.Build())
}

func verifyCertificate(c *gin.Context) {
	var payload entities.PublicCertificate
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// verify the access token and application id
	ok, err := payload.VerifyAccessTokenAndAppId()
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid certificate"})
		return
	}

	// verify the certificate id and createdAt
	ok, err = payload.VerifyCertificateIdAndCreatedAt()
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid certificate"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "certificate verified"})
}

type LoginPayload struct {
	Email string `binding:"required,email"`
	Password string  `binding:"required,min=8,max=32"`
}

func login(c *gin.Context) {
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_db := db.GetDB()
	var company entities.Company
	err := _db.Where("email = ?", payload.Email).First(&company).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !company.VerifyPassword(payload.Password) { 
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid credentials"})
		return
	}

	// generate the token
	token, err := _jwt.Token(map[string]int{
		"id": int(company.ID),
	}, 24 * time.Hour)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}