package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade model definition
type Trade struct {
	ID           uint            `gorm:"primary_key"`
	CreatedAt    time.Time       `gorm:"not null"`
	Date         time.Time       `gorm:"not null"`
	Asset        string          `gorm:"not null"`
	Action       string          `gorm:"not null"`
	Quantity     decimal.Decimal `gorm:"type:decimal;not null"`
	BaseCurrency string          `gorm:"not null"`
	BasePrice    decimal.Decimal `gorm:"type:decimal;not null"`
	BaseFee      decimal.Decimal `gorm:"type:decimal;not null"`
	FileID       uint            ``
	ReportID     uint            `gorm:"not null"`
}

// SaveTrade stores the Trade and returns its ID
func (db *DB) SaveTrade(t *Trade) (uint, error) {
	if t.FileID > 0 {
		c := db.Create(t)

		if c.Error != nil {
			return 0, c.Error
		}

		return c.Value.(*Trade).ID, nil
	}

	// manually insert null in nullable foreign key file_id
	q := "INSERT into trades (created_at, date, asset, action, quantity, base_currency, base_price, base_fee, report_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id"
	c := db.Raw(q, time.Now(), t.Date, t.Asset, t.Action, t.Quantity, t.BaseCurrency, t.BasePrice, t.BaseFee, t.ReportID)

	if c.Error != nil {
		return 0, c.Error
	}

	var tid uint
	err := c.Row().Scan(&tid)

	return tid, err
}

// GetTrade returns trade by ID
func (db *DB) GetTrade(id uint) (*Trade, error) {
	t := &Trade{ID: id}
	err := db.First(t).Error
	return t, err
}

// DeleteTrade by ID
func (db *DB) DeleteTrade(id uint) error {
	return db.Delete(&Trade{ID: id}).Error
}

// GetReportTrades returns the trades for the report ID
func (db *DB) GetReportTrades(rid uint) (trades []*Trade, err error) {
	err = db.Where(&Trade{ReportID: rid}).Find(&trades).Error
	return
}

// GetManualTrades returns the trades for the report ID with no associated File
func (db *DB) GetManualTrades(rid uint) (trades []*Trade, err error) {
	db.Raw("SELECT * FROM trades WHERE report_id=? AND file_id IS NULL", rid).Scan(&trades)
	return
}
