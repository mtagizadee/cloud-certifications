package migration

import (
	"auth/packages/db"
	"auth/packages/entities"
	"fmt"
)

func Migrate() error {
	_db := db.GetDB();

	err := _db.AutoMigrate(&entities.Company{}, &entities.App{}, &entities.Certificate{}, &entities.Integration{}, &entities.Connection{})
	if err != nil {
		return err
	}
	fmt.Println("Database migrated.")

	return nil
}