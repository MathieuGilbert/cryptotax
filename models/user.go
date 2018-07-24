package models

import (
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User holds a database user
type User struct {
	ID           uint      `gorm:"primary_key"`
	CreatedAt    time.Time `gorm:"not null"`
	Email        string    `gorm:"not null;unique_index"`
	Password     string    `gorm:"not null"`
	ConfirmToken string    ``
	Confirmed    bool      ``
}

// GetUser retrieves user by ID
func (db *DB) GetUser(id uint) (*User, error) {
	u := &User{}
	if err := db.First(u, id).Error; err != nil {
		return nil, err
	}

	return u, nil
}

// RegisterUser creates a new user in the DB and returns its ID.
// Encrypts the password using bcrypt.
func (db *DB) RegisterUser(email, password string) (*User, error) {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
	u := &User{
		Email:        email,
		Password:     string(bytes),
		ConfirmToken: Random(128),
		Confirmed:    false,
	}
	dbc := db.Create(u)
	if dbc.Error != nil {
		log.Printf("Error creating user: %v\n%v", u, dbc.Error)
		return nil, dbc.Error
	}
	return dbc.Value.(*User), nil
}

// Authenticate with the provided credentials
func (db *DB) Authenticate(email, password string) (*User, error) {
	// retrieve user by email
	u := &User{Email: email}
	if err := db.First(u).Error; err != nil {
		return nil, err
	}
	// compare hashed with provided plain-text password
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return nil, err
	}
	return u, nil
}

// EmailExists returns whether the email has already been registered
func (db *DB) EmailExists(email string) bool {
	var result struct{ Count int }
	db.Raw("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&result)
	return result.Count > 0
}

// VerifyEmail sets user with matching token to Confirmed
func (db *DB) VerifyEmail(token string) bool {
	if len(token) == 0 {
		return false
	}
	// get user with the token
	u := &User{ConfirmToken: token}
	if err := db.First(u).Error; err != nil {
		return false
	}
	// make sure not already confirmed
	if u.Confirmed {
		return false
	}

	// update fields
	u.Confirmed = true
	u.ConfirmToken = ""
	if err := db.Save(u).Error; err != nil {
		return false
	}

	return true
}
