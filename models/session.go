package models

import (
	"errors"
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
		SessionID: Random(128),
		CSRFToken: Random(256),
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

// Session retrieves a valid, non-expired session by SessionID
func (db *DB) Session(sid string) (*Session, error) {
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
	return db.Save(s).Error
}

// UpgradeSession applies user to the session
func (db *DB) UpgradeSession(s *Session, u *User) error {
	if !s.Valid || s.Expires.Before(time.Now()) {
		return errors.New("session invalid/expired")
	}

	s.CSRFToken = Random(256)
	s.Valid = true
	s.Expires = time.Now().AddDate(1, 0, 0)
	s.UserID = u.ID

	if err := db.Save(s).Error; err != nil {
		return err
	}

	return nil
}
