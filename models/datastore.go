package models

// https://www.alexedwards.net/blog/organising-database-access

import (
	"github.com/jinzhu/gorm"
)

// Datastore implements available DB methods
type Datastore interface {
	//CreateReport(*Report) (uint, error)
	//GetReport(uint) (*Report, error)
	//UpdateReportCurrency(*Report) error
	//SaveFile(*File) (uint, error)
	//GetFile(uint) (*File, error)
	//GetReportFiles(uint) ([]*File, error)
	//GetManualTrades(uint) ([]*Trade, error)
	//DeleteFile(uint) error
	//SaveTrade(*Trade) (uint, error)
	//GetTrade(uint) (*Trade, error)
	//DeleteTrade(uint) error
	//BeginTransaction() *DB
	NewSession(*User) (*Session, error)
	UpgradeSession(*Session, *User) error
	GetSession(string) (*Session, error)
	KillSession(*Session) error
	GetUser(uint) (*User, error)
	EmailExists(string) bool
	RegisterUser(string, string) (*User, error)
	Authenticate(string, string) (*User, error)
	VerifyEmail(string) bool
}

// DB wraps gorm.DB
type DB struct {
	*gorm.DB
}

// BeginTransaction starts a new transaction
func (db *DB) BeginTransaction() *DB {
	return &DB{db.Begin()}
}
