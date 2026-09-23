package database

import (
	"agent-bot/internal/config"
	"agent-bot/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	dbPath := config.Get().DatabasePath
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	_ = db.AutoMigrate(&model.User{})
	DB = db

	return nil
}