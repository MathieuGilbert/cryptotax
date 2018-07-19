package database

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // db driver
	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
	gormigrate "gopkg.in/gormigrate.v1"
)

// Migrate models with gorm
func Migrate() {
	db, err := gorm.Open("postgres", "host=localhost port=5432 user=cryptotax dbname=cryptotax_dev password=password!@# sslmode=disable")
	if err != nil {
		panic(fmt.Errorf("failed to connect database: %v", err))
	}
	if err = db.DB().Ping(); err != nil {
		panic(fmt.Errorf("unable to ping database: %v", err))
	}
	db.LogMode(true)
	defer db.Close()

	if err = start(db); err != nil {
		panic(fmt.Errorf("Could not migrate: %v", err))
	}
	fmt.Println("Migration ran successfully")
}

// IDs are timestamps from command line:
// > date +"%Y%m%d%H%M%S"
func start(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// create reports table
		{
			ID: "20180628150229",
			Migrate: func(tx *gorm.DB) error {
				type Report struct {
					ID        uint      `gorm:"primary_key"`
					CreatedAt time.Time `gorm:"not null"`
					Currency  string    `gorm:"not null"`
					Trade     []models.Trade
				}
				return tx.CreateTable(&Report{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("reports").Error
			},
		},
		// create files table
		{
			ID: "20180628150520",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					ID        uint      `gorm:"primary_key"`
					CreatedAt time.Time `gorm:"not null"`
					Name      string    `gorm:"not null"`
					Source    string    `gorm:"not null"`
					Data      []byte    `gorm:"type:bytea;not null"`
				}
				return tx.CreateTable(&File{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("files").Error
			},
		},
		// create assets table
		{
			ID: "20180628150711",
			Migrate: func(tx *gorm.DB) error {
				type Asset struct {
					ID        uint      `gorm:"primary_key"`
					CreatedAt time.Time `gorm:"not null"`
					Name      string    `gorm:"not null"`
					Symbol    string    `gorm:"not null"`
				}
				return tx.CreateTable(&Asset{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("assets").Error
			},
		},
		// create trades table
		{
			ID: "20180628150738",
			Migrate: func(tx *gorm.DB) error {
				type Trade struct {
					ID           uint            `gorm:"primary_key"`
					CreatedAt    time.Time       `gorm:"not null"`
					Date         time.Time       `gorm:"not null"`
					AssetID      int             `gorm:"not null"`
					Action       string          `gorm:"not null"`
					Quantity     decimal.Decimal `gorm:"type:decimal;not null"`
					BaseCurrency string          `gorm:"not null"`
					BasePrice    decimal.Decimal `gorm:"type:decimal;not null"`
					BaseFee      decimal.Decimal `gorm:"type:decimal;not null"`
					FileID       int             `gorm:"not null"`
					ReportID     int             `gorm:"not null"`
				}
				if err := tx.CreateTable(&Trade{}).Error; err != nil {
					return err
				}
				if err := tx.Model(Trade{}).AddForeignKey("asset_id", "assets(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				if err := tx.Model(Trade{}).AddForeignKey("file_id", "files(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				if err := tx.Model(Trade{}).AddForeignKey("report_id", "reports(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("trades").Error
			},
		},
		// drop Assets table to make it a string field on Trade
		{
			ID: "20180702150954",
			Migrate: func(tx *gorm.DB) error {
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
					FileID       int             `gorm:"not null"`
					ReportID     int             `gorm:"not null"`
				}

				if err := tx.Model(&Trade{}).RemoveForeignKey("asset_id", "assets(id)").Error; err != nil {
					return err
				}
				if err := tx.DropTable("assets").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("asset_id").Error; err != nil {
					return err
				}
				return tx.AutoMigrate(&Trade{}).Error
			},
		},
		// change File.Data to store MD5 of file instead of actual file
		{
			ID: "20180702162049",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					ID        uint      `gorm:"primary_key"`
					CreatedAt time.Time `gorm:"not null"`
					Name      string    `gorm:"not null"`
					Source    string    `gorm:"not null"`
					Hash      []byte    `gorm:"type:bytea;not null"`
				}

				if err := tx.Model(&File{}).DropColumn("data").Error; err != nil {
					return err
				}
				return tx.AutoMigrate(&File{}).Error
			},
		},
		// Drop not null constraint on Trade.FileID
		{
			ID: "20180703091201",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec("ALTER TABLE trades ALTER COLUMN file_id DROP NOT NULL").Error
			},
		},
		// Add unique index on file hash
		{
			ID: "20180703122730",
			Migrate: func(tx *gorm.DB) error {
				type File struct{}
				return db.Model(&File{}).AddUniqueIndex("idx_file_hash", "hash").Error
			},
			Rollback: func(tx *gorm.DB) error {
				return db.RemoveIndex("idx_file_hash").Error
			},
		},
		// Create sessions table
		{
			ID: "20180703174917",
			Migrate: func(tx *gorm.DB) error {
				type Session struct {
					ID        uint      `gorm:"primary_key"`
					CreatedAt time.Time `gorm:"not null"`
					UpdatedAt time.Time `gorm:"not null"`
					SessionID string    `gorm:"not null"`
					CSRFToken string    `gorm:"not null"`
					Valid     bool      `gorm:"not null"`
					Expires   time.Time `gorm:"not null"`
					ReportID  uint      ``
				}
				if err := tx.CreateTable(&Session{}).Error; err != nil {
					return err
				}
				if err := tx.Model(Session{}).AddForeignKey("report_id", "reports(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Session struct{}
				if err := tx.Model(&Session{}).RemoveForeignKey("report_id", "reports(id)").Error; err != nil {
					return err
				}
				return tx.DropTable("sessions").Error
			},
		},
		// Add FK from files to reports
		{
			ID: "20180704155658",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					ID        uint      `gorm:"primary_key"`
					CreatedAt time.Time `gorm:"not null"`
					Name      string    `gorm:"not null"`
					Source    string    `gorm:"not null"`
					Hash      []byte    `gorm:"type:bytea;not null"`
					ReportID  uint      `gorm:"not null"`
				}
				if err := tx.AutoMigrate(&File{}).Error; err != nil {
					return err
				}
				return tx.Model(File{}).AddForeignKey("report_id", "reports(id)", "RESTRICT", "RESTRICT").Error
			},
			Rollback: func(tx *gorm.DB) error {
				type File struct{}
				if err := tx.Model(&File{}).RemoveForeignKey("report_id", "reports(id)").Error; err != nil {
					return err
				}
				return tx.Model(&File{}).DropColumn("report_id").Error
			},
		},
		// Better unique index on file hash + report_id
		{
			ID: "20180704155820",
			Migrate: func(tx *gorm.DB) error {
				type File struct{}
				if err := db.RemoveIndex("idx_file_hash").Error; err != nil {
					return err
				}
				return db.Model(&File{}).AddUniqueIndex("idx_file_hash_report_id", "hash", "report_id").Error
			},
			Rollback: func(tx *gorm.DB) error {
				type File struct{}
				if err := db.RemoveIndex("idx_file_hash_report_id").Error; err != nil {
					return err
				}
				return db.Model(&File{}).AddUniqueIndex("idx_file_hash", "hash").Error
			},
		},
		// Cascade delete file trades
		{
			ID: "20180711164029",
			Migrate: func(tx *gorm.DB) error {
				type Trade struct{}
				if err := tx.Model(&Trade{}).RemoveForeignKey("file_id", "files(id)").Error; err != nil {
					return err
				}
				return tx.Model(&Trade{}).AddForeignKey("file_id", "files(id)", "CASCADE", "RESTRICT").Error
			},
			Rollback: func(tx *gorm.DB) error {
				type Trade struct{}
				if err := tx.Model(&Trade{}).RemoveForeignKey("file_id", "files(id)").Error; err != nil {
					return err
				}
				return tx.Model(&Trade{}).AddForeignKey("file_id", "files(id)", "RESTRICT", "RESTRICT").Error
			},
		},
	})

	return m.Migrate()
}
