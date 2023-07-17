package entities

import (
	"auth/packages/_jwt"
	"auth/packages/db"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/**
CERTIFICATE
accessToken -- {companyId}
applicationId
certificatedId
createdAt
*/

type Certificate struct {
	ID string `gorm:"primaryKey"`
	AppID int
	CreatedAt time.Time 
	UpdatedAt time.Time
  DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (cert *Certificate) GenerateId() {
	cert.ID = uuid.New().String()
}

func (cert Certificate) GenerateAccessToken(companyId int) (string, error) {
	return _jwt.Token(map[string]int{
		"CompanyId": companyId,
	})
}

type PublicCertificate struct  {
	AccessToken string `binding:"required"`
	ApplicationId int `binding:"required"`
	CertificateId string `binding:"required"`
	CreatedAt time.Time `binding:"required"`
} 

func (cert PublicCertificate) VerifyAccessTokenAndAppId() (bool, error) {
	claims, err := _jwt.Claims(cert.AccessToken)
	if err != nil {
		return false, err
	}

	// check if the company is the owner of the certificate
	_db := db.GetDB()
	var company Company
	err = _db.Model(&company).Preload("Apps").Where("id = ?", claims.CustomClaims["CompanyId"]).First(&company).Error
	if err != nil { // company not found
		return false, err
	}

	// check if company owns the application
	if !company.HasApp(cert.ApplicationId) {
		return false, errors.New("company does not own the application")
	}	

	return true, nil
}

func (cert PublicCertificate) VerifyCertificateIdAndCreatedAt() (bool, error) {
	_db := db.GetDB()
	var certificate Certificate
	err := _db.Model(&certificate).Where("id = ?", cert.CertificateId).First(&certificate).Error
	if err != nil { // certificate not found
		return false, err
	}

	return cert.ApplicationId == certificate.AppID && cert.CreatedAt == certificate.CreatedAt, nil
}