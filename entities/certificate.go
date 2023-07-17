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

type CertificateBuilder struct {
	certificate Certificate
	public PublicCertificate
}

func (builder *CertificateBuilder) Prepare() error {
	_db := db.GetDB()
	err := _db.Create(&builder.certificate).Error // create the certificate and save it to the database
	if err != nil {
		return err
	}
	
	return nil
}

func (builder *CertificateBuilder) Build() *PublicCertificate {
	builder.public.ApplicationId = builder.certificate.AppID
	builder.public.CertificateId = builder.certificate.ID
	builder.public.CreatedAt = builder.certificate.CreatedAt
	
	return &builder.public
}

func (builder *CertificateBuilder) GenerateId() *CertificateBuilder {
	builder.certificate.ID = uuid.New().String()
	return builder
}

func (builder *CertificateBuilder) SetAppId(appId int) *CertificateBuilder {
	builder.certificate.AppID = appId
	return builder
}

func (builder *CertificateBuilder) GenerateAccessToken(companyId int) (*CertificateBuilder,error) {
	token, err := _jwt.Token(map[string]int{
		"CompanyId": companyId,
	})
	if err != nil {
		return nil, err
	}

	builder.public.AccessToken = token
	return builder, nil
}

func NewCertificate() *CertificateBuilder {
	return &CertificateBuilder{
		certificate: Certificate{},
		public: PublicCertificate{},
	}
}
