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
				if err := tx.Model(&Trade{}).AddForeignKey("asset_id", "assets(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).AddForeignKey("file_id", "files(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).AddForeignKey("report_id", "reports(id)", "RESTRICT", "RESTRICT").Error; err != nil {
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
				return tx.Model(&File{}).AddUniqueIndex("idx_file_hash", "hash").Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.RemoveIndex("idx_file_hash").Error
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
				if err := tx.RemoveIndex("idx_file_hash").Error; err != nil {
					return err
				}
				return tx.Model(&File{}).AddUniqueIndex("idx_file_hash_report_id", "hash", "report_id").Error
			},
			Rollback: func(tx *gorm.DB) error {
				type File struct{}
				if err := tx.RemoveIndex("idx_file_hash_report_id").Error; err != nil {
					return err
				}
				return tx.Model(&File{}).AddUniqueIndex("idx_file_hash", "hash").Error
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
		// Add users table
		{
			ID: "20180720172813",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					ID       uint   `gorm:"primary_key"`
					Email    string `gorm:"not null"`
					Password string `gorm:"not null"`
				}
				if err := tx.AutoMigrate(&User{}).Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.DropTable("users").Error
			},
		},
		// Put user in session instead of report
		{
			ID: "20180720172908",
			Migrate: func(tx *gorm.DB) error {
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

				if err := tx.Model(&Session{}).RemoveForeignKey("report_id", "reports(id)").Error; err != nil {
					return err
				}

				if err := tx.Model(&Session{}).DropColumn("report_id").Error; err != nil {
					return err
				}

				if err := tx.AutoMigrate(&Session{}).Error; err != nil {
					return err
				}

				if err := tx.Model(&Session{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Session struct {
					ReportID uint ``
				}

				if err := tx.Model(&Session{}).RemoveForeignKey("user_id", "users(id)").Error; err != nil {
					return err
				}

				if err := tx.Model(&Session{}).DropColumn("user_id").Error; err != nil {
					return err
				}

				if err := tx.AutoMigrate(&Session{}).Error; err != nil {
					return err
				}

				if err := tx.Model(&Session{}).AddForeignKey("report_id", "reports(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}

				return nil
			},
		},
		// Add createdat to users
		{
			ID: "20180720173153",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					ID        uint      `gorm:"primary_key"`
					CreatedAt time.Time `gorm:"not null"`
					Email     string    `gorm:"not null"`
					Password  string    `gorm:"not null"`
				}
				if err := tx.AutoMigrate(&User{}).Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type User struct{}
				if err := tx.Model(&User{}).DropColumn("created_at").Error; err != nil {
					return err
				}

				return nil
			},
		},
		// Make UserID nullable reference by removing the FK
		{
			ID: "20180721082820",
			Migrate: func(tx *gorm.DB) error {
				type Session struct{}
				return tx.Model(&Session{}).RemoveForeignKey("user_id", "users(id)").Error
			},
			Rollback: func(tx *gorm.DB) error {
				type Session struct{}
				return tx.Model(&Session{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").Error
			},
		},
		// make email a unique index on users
		{
			ID: "20180721184203",
			Migrate: func(tx *gorm.DB) error {
				type User struct{}
				return tx.Model(&User{}).AddUniqueIndex("idx_user_email", "email").Error
			},
			Rollback: func(tx *gorm.DB) error {
				type User struct{}
				return tx.Model(&User{}).RemoveIndex("idx_user_email").Error
			},
		},
		// add email verification token to users
		{
			ID: "20180722153137",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					ConfirmToken string ``
					Confirmed    bool   ``
				}
				return tx.AutoMigrate(&User{}).Error
			},
			Rollback: func(tx *gorm.DB) error {
				type User struct{}
				if err := tx.Model(&User{}).DropColumn("confirm_token").Error; err != nil {
					return err
				}
				if err := tx.Model(&User{}).DropColumn("confirmed").Error; err != nil {
					return err
				}
				return nil
			},
		},
		// store file bytes instead of just the hash
		// file associated to a user, not report
		{
			ID: "20180727141736",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					Bytes  []byte `gorm:"type:bytea;not null"`
					UserID uint   `gorm:"not null"`
				}
				if err := tx.AutoMigrate(&File{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).DropColumn("hash").Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).RemoveForeignKey("report_id", "reports(id)").Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).DropColumn("report_id").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type File struct {
					Hash     []byte `gorm:"type:bytea;not null"`
					ReportID uint   `gorm:"not null"`
				}
				if err := tx.AutoMigrate(&File{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).AddForeignKey("report_id", "reports(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).DropColumn("bytes").Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).RemoveForeignKey("user_id", "users(id)").Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).DropColumn("user_id").Error; err != nil {
					return err
				}
				if err := tx.Model(&File{}).AddUniqueIndex("idx_file_hash", "hash").Error; err != nil {
					return err
				}
				return nil
			},
		},
		// drop reportID from trades
		{
			ID: "20180727165446",
			Migrate: func(tx *gorm.DB) error {
				type Trade struct{}
				if err := tx.Model(&Trade{}).RemoveForeignKey("report_id", "reports(id)").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("report_id").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Trade struct {
					ReportID uint `gorm:"not null"`
				}
				if err := tx.AutoMigrate(&Trade{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).AddForeignKey("report_id", "reports(id)", "RESTRICT", "RESTRICT").Error; err != nil {
					return err
				}
				return nil
			},
		},
		// unique index on files bytes
		{
			ID: "20180727171609",
			Migrate: func(tx *gorm.DB) error {
				type File struct{}
				if err := tx.Model(&File{}).AddUniqueIndex("idx_file_bytes_user_id", "digest(bytes, 'sha1')", "user_id").Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type File struct{}
				if err := tx.Model(&File{}).RemoveIndex("idx_file_bytes_user_id").Error; err != nil {
					return err
				}
				return nil
			},
		},
		// rename trade fields: asset -> currency, quantity -> amount, base_price -> base_amount, base_fee -> fee_amount
		// add fee_currency
		{
			ID: "20180728132644",
			Migrate: func(tx *gorm.DB) error {
				type Trade struct {
					Currency    string          `gorm:"not null"`
					Amount      decimal.Decimal `gorm:"type:decimal;not null"`
					BaseAmount  decimal.Decimal `gorm:"type:decimal;not null"`
					FeeAmount   decimal.Decimal `gorm:"type:decimal;not null"`
					FeeCurrency string          `gorm:"not null"`
				}
				if err := tx.Model(&Trade{}).DropColumn("asset").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("quantity").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("base_price").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("base_fee").Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&Trade{}).Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Trade struct {
					Asset     string          `gorm:"not null"`
					Quantity  decimal.Decimal `gorm:"type:decimal;not null"`
					BasePrice decimal.Decimal `gorm:"type:decimal;not null"`
					BaseFee   decimal.Decimal `gorm:"type:decimal;not null"`
				}
				if err := tx.Model(&Trade{}).DropColumn("currency").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("amount").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("base_amount").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("fee_amount").Error; err != nil {
					return err
				}
				if err := tx.Model(&Trade{}).DropColumn("fee_currency").Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&Trade{}).Error; err != nil {
					return err
				}
				return nil
			},
		},
	})

	return m.Migrate()
}
