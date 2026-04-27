package model

import (
	"time"

	"gorm.io/gorm"
)

type Platform struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"not null" json:"name"`
	SubType string `json:"sub_type"` // "server", "client", "primary", "backup"
}

type Group struct {
	ID     uint     `gorm:"primaryKey" json:"id"`
	Name   string   `gorm:"unique;not null" json:"name"`
	Order  int      `json:"order"` // Display order
	People []Person `json:"people,omitempty"`
}

type Person struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"not null" json:"name"`
	Phone   string `json:"phone"`
	GroupID uint   `json:"group_id"`
	Group   Group  `json:"-"`
}

type Shift struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Date       string    `gorm:"index;not null;type:date" json:"date"` // Format: YYYY-MM-DD
	GroupID    uint      `json:"group_id"`
	PlatformID uint      `json:"platform_id"`
	PersonID   uint      `json:"person_id"`
	Person     Person    `json:"person"`
	Group      Group     `json:"group"`
	Platform   Platform  `json:"platform"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Migrate checks and creates tables
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Group{}, &Platform{}, &Person{}, &Shift{})
}
