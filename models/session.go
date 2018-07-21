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
	UserID    uint      ``
}

// NewSession creates a new session
func (db *DB) NewSession(u *User) (*Session, error) {
	session := &Session{
		SessionID: random(128),
		CSRFToken: random(256),
		Valid:     true,
		Expires:   time.Now().AddDate(1, 0, 0),
	}
	if u != nil {
		session.UserID = u.ID
	}

	dbc := db.Create(session)
	if dbc.Error != nil {
		return nil, errors.New("unable to create session")
	}

	return dbc.Value.(*Session), nil
}

// GetSession retrieves a valid, non-expired session by SessionID
func (db *DB) GetSession(sid string) (*Session, error) {
	s := &Session{}

	if err := db.Where(&Session{SessionID: sid}).First(s).Error; err != nil {
		return nil, errors.New("session not found")
	}
	if !s.Valid || s.Expires.Before(time.Now()) {
		return nil, errors.New("session invalid/expired")
	}

	return s, nil
}

// KillSession invalidates a session
func (db *DB) KillSession(s *Session) error {
	s.Valid = false

	if err := db.Save(s).Error; err != nil {
		return errors.New("unable to save session")
	}

	return nil
}

// UpgradeSession applies user to the session
func (db *DB) UpgradeSession(s *Session, u *User) error {
	if !s.Valid || s.Expires.Before(time.Now()) {
		return errors.New("session invalid/expired")
	}

	s.CSRFToken = random(256)
	s.Valid = true
	s.Expires = time.Now().AddDate(1, 0, 0)
	s.UserID = u.ID

	if err := db.Save(s).Error; err != nil {
		return errors.New("unable to save session")
	}

	return nil
}

// Random returns a random, url safe value of the bit length passed in
// https://tech.townsourced.com/post/anatomy-of-a-go-web-app-authentication/
func random(bits int) string {
	result := make([]byte, bits/8)
	_, err := io.ReadFull(rand.Reader, result)
	if err != nil {
		panic(fmt.Sprintf("Error generating random values: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(result)
}
