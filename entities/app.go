package entities

import "gorm.io/gorm"

type App struct {
	gorm.Model
	Name string `gorm:"not null" binding:"required,min=2"`
	CompanyID int
	Certificates []Certificate `gorm:"foreignkey:AppID"`
}