package entities

import (
	"crypto/sha256"
	"fmt"

	"gorm.io/gorm"
)

type Company struct { 
	gorm.Model
	Name string `gorm:"not null" binding:"required,min=2"`
	Email string `gorm:"unique;not null" binding:"required,email"`
	Password string `gorm:"not null" binding:"required,min=4,max=16"`
	Apps []App `gorm:"foreignkey:CompanyID"`
}

func (c *Company) HasApp(appId int) (bool) {
	for _, app := range c.Apps {
		if app.ID == uint(appId) {
			return true
		}
	}
	return false
}

func (c *Company) HashPassword() {
	hash := sha256.Sum256([]byte(c.Password))
	c.Password = fmt.Sprintf("%x", hash)
}

func (c *Company) VerifyPassword(password string) bool {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash) == c.Password
}