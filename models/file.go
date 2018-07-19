package models

import (
	"time"
)

// File model definition
type File struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `gorm:"not null"`
	Name      string    `gorm:"not null"`
	Source    string    `gorm:"not null"`
	Hash      []byte    `gorm:"type:bytea;not null"`
	ReportID  uint      `gorm:"not null"`
}

// SaveFile stores the file metadata and returns its ID
func (db *DB) SaveFile(file *File) (uint, error) {
	dbc := db.Create(file)
	if dbc.Error != nil {
		return 0, dbc.Error
	}
	return dbc.Value.(*File).ID, nil
}

// GetReportFiles retrives all files for the report ID
func (db *DB) GetReportFiles(rid uint) (files []*File, err error) {
	err = db.Where(&File{ReportID: rid}).Find(&files).Error
	return
}

// GetFile returns file by ID
func (db *DB) GetFile(id uint) (*File, error) {
	file := &File{ID: id}
	err := db.First(file).Error
	return file, err
}

// DeleteFile and cascade delete associate trades
func (db *DB) DeleteFile(id uint) error {
	return db.Delete(&File{ID: id}).Error
}
