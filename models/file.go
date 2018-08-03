package models

import (
	"errors"
	"time"
)

// File model definition
type File struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `gorm:"not null"`
	Name      string    `gorm:"not null"`
	Source    string    `gorm:"not null"`
	Bytes     []byte    `gorm:"type:bytea;not null"`
	UserID    uint      `gorm:"not null"`
}

// SaveFile stores the file metadata and returns its ID
func (db *DB) SaveFile(file *File) (uint, error) {
	dbc := db.Create(file)
	if dbc.Error != nil {
		return 0, dbc.Error
	}
	return dbc.Value.(*File).ID, nil
}

// GetFile returns file by ID
func (db *DB) GetFile(id uint) (*File, error) {
	file := &File{ID: id}
	err := db.First(file).Error
	return file, err
}

// GetFiles returns a user's files
func (db *DB) GetFiles(uid uint) ([]*File, error) {
	var fs []*File
	err := db.Select("id, created_at, name, source").Where(&File{UserID: uid}).Order("created_at asc").Find(&fs).Error
	return fs, err
}

// DeleteFile and cascade delete associate trades
func (db *DB) DeleteFile(id uint, uid uint) error {
	q := db.Exec("DELETE FROM files WHERE id = ? AND user_id = ?", id, uid)
	if deleted := q.RowsAffected == 1; !deleted {
		return errors.New("unable to delete file")
	}
	return q.Error
}
