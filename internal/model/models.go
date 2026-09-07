package model

type User struct {
	ID	   uint   `gorm:"primaryKey"`
	DiscordID string `gorm:"uniqueIndex"`
	Username  string
	Token	string

	CreatedAt  int64
	UpdatedAt  int64
	DeletedAt  int64
}