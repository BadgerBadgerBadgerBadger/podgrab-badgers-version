package db

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"
)

// Base is a struct that contains common fields for all
// database tables.
type Base struct {
	ID        string `sql:"type:uuid;primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// BeforeCreate is a Gorm hook run before inserting a new
// entry into the database.
func (base *Base) BeforeCreate(tx *gorm.DB) error {
	tx.Statement.SetColumn("ID", uuid.NewV4().String())
	return nil
}
