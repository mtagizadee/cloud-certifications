package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func setDB(database *gorm.DB) {
	db = database
}

func getDB() *gorm.DB {
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

	err = db.AutoMigrate(&company{}, &app{}, &certificate{})
	if err != nil {
		panic("Could not migrate the database.")
	}
	fmt.Println("Database migrated.")
	
	companies := v1.Group("/companies")
	companies.POST("/", addCompany)
	companies.GET("/:id", getCompanyById)
	companies.POST("/:id/apps", addApp);
	companies.POST(":id/apps/:appId/certificates", addCertificate)

	certificates := v1.Group("/certificates")
	certificates.POST("/verify", verifyCertificate)
	
	

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

func (c *company) hasApp(appId int) (bool) {
	for _, app := range c.Apps {
		if app.ID == uint(appId) {
			return true
		}
	}
	return false
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

		db := getDB()
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

	db := getDB()
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
	Certificates []certificate `gorm:"foreignkey:AppID"`
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

	db := getDB()
	err = db.Create(&application).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, application)
}

type certificate struct {
	ID string `gorm:"primaryKey"`
	AppID int
	CreatedAt time.Time 
	UpdatedAt time.Time
  DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (cert *certificate) generateId() {
	cert.ID = uuid.New().String()
}

type jwtCertificatePayload struct {
	jwt.StandardClaims
	CustomClaims map[string]int
}

func getSecretKey() []byte {
	return []byte("supersecret") // IMPORTANT: replace it after the development
}

func (cert certificate) generateAccessToken(companyId int) (string, error) {
	claims := jwtCertificatePayload{
			StandardClaims: jwt.StandardClaims{
			 // set token lifetime in timestamp
			 ExpiresAt: time.Now().Add(28 * 24 * time.Hour).Unix(),
		},
		// add custom claims like user_id or email, 
		// it can vary according to requirements
		CustomClaims: map[string]int{
			"CompanyId": companyId,
		},
 }

 // generate a string using claims and HS256 algorithm
 tokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

 // sign the generated key using secretKey
 token, err := tokenString.SignedString(getSecretKey()) 

 return token, err
}


/**
	CERTIFICATE
	accessToken -- {companyId}
	applicationId
	certificatedId
	createdAt
*/

type publicCertificate struct  {
	AccessToken string `binding:"required"`
	ApplicationId int `binding:"required"`
	CertificateId string `binding:"required"`
	CreatedAt time.Time `binding:"required"`
} 

func (cert publicCertificate) verifyAccessTokenAndAppId() (bool, error) {
	claims := &jwtCertificatePayload{}

	_, err := jwt.ParseWithClaims(cert.AccessToken, claims, func(token *jwt.Token) (interface{}, error) {
			return getSecretKey(), nil
	})

	if err != nil { // expired or invalid token
		return false, err
	}

	// check if the company is the owner of the certificate
	db := getDB()
	var company company
	err = db.Model(&company).Preload("Apps").Where("id = ?", claims.CustomClaims["CompanyId"]).First(&company).Error
	if err != nil { // company not found
		return false, err
	}

	// check if company owns the application
	if !company.hasApp(cert.ApplicationId) {
		return false, errors.New("company does not own the application")
	}	

	return true, nil
}

func (cert publicCertificate) verifyCertificateIdAndCreatedAt() (bool, error) {
	db := getDB()
	var certificate certificate
	err := db.Model(&certificate).Where("id = ?", cert.CertificateId).First(&certificate).Error
	if err != nil { // certificate not found
		return false, err
	}

	return cert.ApplicationId == certificate.AppID && cert.CreatedAt == certificate.CreatedAt, nil
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

	fmt.Println(appId, companyId)
	
	
	var certificate certificate
	accessToken, err := certificate.generateAccessToken(companyId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(accessToken)

	db := getDB();
	// setup the certificate
	certificate.AppID = appId
	certificate.generateId()
	err = db.Create(&certificate).Error // create the certificate and save it to the database
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, &publicCertificate{
		AccessToken: accessToken,
		CertificateId: certificate.ID,
		ApplicationId: certificate.AppID,
		CreatedAt: certificate.CreatedAt,
	})
}

func verifyCertificate(c *gin.Context) {
	var payload publicCertificate
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// verify the access token and application id
	ok, err := payload.verifyAccessTokenAndAppId()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid certificate"})
		return
	}

	// verify the certificate id and createdAt
	ok, err = payload.verifyCertificateIdAndCreatedAt()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid certificate"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "certificate verified"})
}