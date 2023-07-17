package entities

type Integration struct {
	ID string `gorm:"primaryKey,size:124"` // uuid
	Connections []Connection `gorm:"foreignKey:IntegrationID"` // gorm:"foreignkey:CompanyID 
}

type Connection struct {
	ID int `gorm:"primaryKey" binding:"required"`
	IntegrationID string `gorm:"size:124" binding:"required"`
}

func (integration Integration) HasConnection(connectionId int) bool {
	for _, connection := range integration.Connections {
		if connection.ID == connectionId {
			return true
		}
	}

	return false
}
