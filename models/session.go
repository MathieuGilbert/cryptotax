package models

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

// Session defines a user's session
type Session struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
	SessionID string    `gorm:"not null"`
	CSRFToken string    `gorm:"not null"`
	Valid     bool      `gorm:"not null"`
	Expires   time.Time `gorm:"not null"`
	ReportID  uint      ``
}

// NewSession generates a new session
func (db *DB) NewSession(r *Report) (*Session, error) {
	session := &Session{
		SessionID: Random(128),
		CSRFToken: Random(256),
		Valid:     true,
		Expires:   time.Now().AddDate(1, 0, 0),
		ReportID:  r.ID,
	}
	dbc := db.Create(session)
	if dbc.Error != nil {
		return nil, dbc.Error
	}
	return dbc.Value.(*Session), nil
}

// GetSession retrieves a valid, non-expired session by SessionID
func (db *DB) GetSession(sid string) (*Session, error) {
	s := &Session{}
	err := db.Where(&Session{SessionID: sid}).First(s).Error
	if err != nil {
		return nil, err
	}
	if !s.Valid || s.Expires.Before(time.Now()) {
		return nil, errors.New("Session invalid")
	}
	return s, nil
}

// KillSession invalidates a session
func (db *DB) KillSession(sid string) error {
	s := &Session{}
	err := db.Where(&Session{SessionID: sid}).First(s).Error
	if err != nil {
		return err
	}
	s.Valid = false
	db.Save(s)
	return nil
}

// Random returns a random, url safe value of the bit length passed in
// https://tech.townsourced.com/post/anatomy-of-a-go-web-app-authentication/
func Random(bits int) string {
	result := make([]byte, bits/8)
	_, err := io.ReadFull(rand.Reader, result)
	if err != nil {
		panic(fmt.Sprintf("Error generating random values: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(result)
}
