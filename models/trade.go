package models

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// Trade model definition
type Trade struct {
	ID           uint            `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time       `gorm:"not null" json:"created_at"`
	Date         time.Time       `gorm:"not null" json:"date"`
	Action       string          `gorm:"not null" json:"action"`
	Amount       decimal.Decimal `gorm:"type:decimal;not null" json:"amount"`
	Currency     string          `gorm:"not null" json:"currency"`
	BaseAmount   decimal.Decimal `gorm:"type:decimal;not null" json:"base_amount"`
	BaseCurrency string          `gorm:"not null" json:"base_currency"`
	FeeAmount    decimal.Decimal `gorm:"type:decimal;not null" json:"fee_amount"`
	FeeCurrency  string          `gorm:"not null" json:"fee_currency"`
	FileID       uint            `json:"file_id"`
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

	// handle nullable foreign key file_id
	var fid sql.NullInt64
	if t.FileID != 0 {
		fid = sql.NullInt64{Int64: int64(t.FileID)}
	}

	q := "INSERT into trades (created_at, date, action, currency, amount, base_currency, base_amount, fee_amount, fee_currency, file_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id"
	c := db.Raw(q, time.Now(), t.Date, t.Action, t.Currency, t.Amount, t.BaseCurrency, t.BaseAmount, t.FeeAmount, t.FeeCurrency, fid)

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

// GetFileTrades returns trades for the file id and user id
func (db *DB) GetFileTrades(fid uint, uid uint) ([]*Trade, error) {
	if f, err := db.GetFile(fid); err != nil || f.UserID != uid {
		return nil, err
	}
	var ts []*Trade
	err := db.Where(&Trade{FileID: fid}).Find(&ts).Error
	return ts, err
}

// GetManualTrades returns the trades for the report ID with no associated File
func (db *DB) GetManualTrades(rid uint) (trades []*Trade, err error) {
	db.Raw("SELECT * FROM trades WHERE report_id=? AND file_id IS NULL", rid).Scan(&trades)
	return
}
