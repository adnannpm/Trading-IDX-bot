package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	db, err := gorm.Open(sqlite.Open("nusa.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	DB = db

	return nil
}