package models

import "time"

// User holds a database user
type User struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `gorm:"not null"`
	Email     string    `gorm:"not null"`
	Password  string    `gorm:"not null"`
}

// CreateUser creates a new user and returns its ID
func (db *DB) CreateUser(u *User) (uint, error) {
	dbc := db.Create(u)
	if dbc.Error != nil {
		return 0, dbc.Error
	}
	return dbc.Value.(*User).ID, nil
}

// GetUser retrieves user by ID
func (db *DB) GetUser(id uint) (*User, error) {
	u := &User{}
	if err := db.First(u, id).Error; err != nil {
		return nil, err
	}

	return u, nil
}

// Active indicates whether user account is activated/confirmed
func (u *User) Active() bool {
	return true
}
