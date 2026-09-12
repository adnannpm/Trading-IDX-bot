package database

import (
	"agent-bot/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	db, err := gorm.Open(sqlite.Open("nusa.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	_ = db.AutoMigrate(&model.User{})
	DB = db

	return nil
}