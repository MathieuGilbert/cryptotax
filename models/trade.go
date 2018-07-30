package models

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// Trade model definition
type Trade struct {
	ID           uint            `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time       `gorm:"not null" json:"createdAt"`
	Date         time.Time       `gorm:"not null" json:"date"`
	Action       string          `gorm:"not null" json:"action"`
	Amount       decimal.Decimal `gorm:"type:decimal;not null" json:"amount"`
	Currency     string          `gorm:"not null" json:"currency"`
	BaseAmount   decimal.Decimal `gorm:"type:decimal;not null" json:"baseAmount"`
	BaseCurrency string          `gorm:"not null" json:"baseCurrency"`
	FeeAmount    decimal.Decimal `gorm:"type:decimal;not null" json:"feeAmount"`
	FeeCurrency  string          `gorm:"not null" json:"feeCurrency"`
	FileID       uint            `json:"fileId"`
	UserID       uint            `gorm:"not null" json:"userId"`
}

// SaveTrade stores the Trade and returns its ID
func (db *DB) SaveTrade(t *Trade) (*Trade, error) {
	// handle nullable foreign key file_id
	fid := sql.NullInt64{Int64: int64(t.FileID), Valid: t.FileID > 0}

	q := "INSERT into trades (created_at, date, action, currency, amount, base_currency, base_amount, fee_amount, fee_currency, file_id, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id"
	c := db.Raw(q, time.Now(), t.Date, t.Action, t.Currency, t.Amount, t.BaseCurrency, t.BaseAmount, t.FeeAmount, t.FeeCurrency, fid, t.UserID)
	if c.Error != nil {
		return nil, c.Error
	}

	var tid uint
	if err := c.Row().Scan(&tid); err != nil {
		return nil, err
	}

	return db.GetTrade(tid)
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

// GetManualTrades returns the trades for the user ID with no associated File
func (db *DB) GetManualTrades(uid uint) (trades []*Trade, err error) {
	err = db.Raw("SELECT * FROM trades WHERE user_id=? AND file_id IS NULL", uid).Scan(&trades).Error
	return
}
