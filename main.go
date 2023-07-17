package main

import (
	"auth/packages/db"
	"auth/packages/entities"
	"auth/packages/migration"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	connections := v1.Group("/connections")
	connections.POST("/", addConnection)
	connections.POST("/verify", verifyConnection)

	integrations := v1.Group("/integrations")
	integrations.GET("/", getIntergrations)
	
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

func getIntergrations(c *gin.Context) {
	_db := db.GetDB()
	var integrations []entities.Integration
	err := _db.Preload("Connections").Find(&integrations).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, integrations)
}

func addInegration() (entities.Integration, error) {
	_db := db.GetDB()
	integration := entities.Integration{ ID: uuid.New().String() }
			
	err := _db.Create(&integration).Error
	if err != nil {
		return integration, err
	}

	return integration, nil
}

func addConnection(c *gin.Context) {
	_db := db.GetDB()
	var integrations []entities.Integration
	err := _db.Preload("Connections").Find(&integrations).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var _integration entities.Integration
	if len(integrations) > 0 { // if there are integrations in the database => use the last one
		lastIntegration := integrations[len(integrations) - 1]
		if len(lastIntegration.Connections) >= 10 { // if the last integration has 10 connections => create new one
			integration, err := addInegration()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			_integration = integration
		} else { // if the last integration has less than 10 connections => use it
			_integration = lastIntegration
		}
	} else { // if no integrations in the database => create one
		integration, err := addInegration()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_integration = integration
	}

	var connection entities.Connection
	connection.IntegrationID = _integration.ID
	// create connection 
	err = _db.Create(&connection).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, connection)
}


func verifyConnection(c *gin.Context) {
	var connection entities.Connection
	if err := c.ShouldBindJSON(&connection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_db := db.GetDB()
	// find integration by id and join with connections
	var integration entities.Integration
	err := _db.Model(&entities.Integration{}).Preload("Connections").Where("id = ?", connection.IntegrationID).First(&integration).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !integration.HasConnection(connection.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid connection"})
		return
	} 
	
	c.JSON(http.StatusOK, gin.H{"message": "connection verified"})
} 
