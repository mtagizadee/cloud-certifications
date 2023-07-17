package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var _db *gorm.DB

func GetDB() *gorm.DB {
	return _db
}

func Connect() error {
	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/authdb?charset=utf8mb4&parseTime=True&loc=Local"))
	_db = db
	if err != nil {
		return err	
	}
	fmt.Println("Connected to the database.")	

	return nil
}


