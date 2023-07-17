package entities

type Integration struct {
	ID string `gorm:"primaryKey,size:124"` // uuid
	Connections []Connection `gorm:"foreignKey:IntegrationID"` // gorm:"foreignkey:CompanyID 
}

type Connection struct {
	ID int `gorm:"primaryKey"`
	IntegrationID string `gorm:"size:124"`
}
