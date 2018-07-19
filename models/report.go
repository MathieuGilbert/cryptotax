package models

import (
	"time"
)

// Report model definition
type Report struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `gorm:"not null"`
	Currency  string    `gorm:"not null"`
	Trades    []*Trade
}

// CreateReport creates a new report and returns its ID
func (db *DB) CreateReport(r *Report) (uint, error) {
	dbc := db.Create(r)
	if dbc.Error != nil {
		return 0, dbc.Error
	}
	return dbc.Value.(*Report).ID, nil
}

// GetReport retrieves a Report by ID, including associated trades
func (db *DB) GetReport(id uint) (*Report, error) {
	r := &Report{}
	if err := db.First(r, id).Error; err != nil {
		return nil, err
	}
	ts, err := db.GetReportTrades(id)
	if err != nil {
		return nil, err
	}
	r.Trades = ts

	return r, nil
}

// UpdateReportCurrency updates the report's currency
func (db *DB) UpdateReportCurrency(r *Report) error {
	q := "UPDATE reports SET currency = ? WHERE id = ?"
	if err := db.Raw(q, r.Currency, r.ID).Error; err != nil {
		return err
	}

	return nil
}
