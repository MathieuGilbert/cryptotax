package models

// https://www.alexedwards.net/blog/organising-database-access

import (
	"github.com/jinzhu/gorm"
)

// Datastore implements available DB methods
type Datastore interface {
	BeginTransaction() *DB
	NewSession(*User) (*Session, error)
	UpgradeSession(*Session, *User) error
	Session(string) (*Session, error)
	KillSession(*Session) error
	GetUser(uint) (*User, error)
	EmailExists(string) bool
	RegisterUser(string, string) (*User, error)
	Authenticate(string, string) (*User, error)
	VerifyEmail(string) bool
	GetFiles(uint) ([]*File, error)
	DeleteFile(uint, uint) error
	GetFileTrades(uint, uint) ([]*Trade, error)
	GetManualTrades(uint) ([]*Trade, error)
	SaveTrade(*Trade) (*Trade, error)
	DeleteTrade(uint, uint) error
	GetUserTrades(uint) ([]*Trade, error)
}

// DB wraps gorm.DB
type DB struct {
	*gorm.DB
}

// BeginTransaction starts a new transaction
func (db *DB) BeginTransaction() *DB {
	return &DB{db.Begin()}
}
