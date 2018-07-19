package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // db driver
	"github.com/mathieugilbert/cryptotax/models"
)

// NewDB creates a new connection
func NewDB(source string) (*models.DB, error) {
	db, err := gorm.Open("postgres", source)
	if err != nil {
		return nil, err
	}

	return &models.DB{DB: db}, nil
}
