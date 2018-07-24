package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"runtime"

	_ "github.com/jinzhu/gorm/dialects/postgres" // db driver
	"github.com/julienschmidt/httprouter"
	"github.com/mathieugilbert/cryptotax/cmd/config"
	"github.com/mathieugilbert/cryptotax/cmd/parsers"
	"github.com/mathieugilbert/cryptotax/database"
	"github.com/mathieugilbert/cryptotax/models"
)

// Env - holds the handlers
type Env struct {
	db models.Datastore
}

// Parser is an interface for exchange-specific parsing logic
type Parser interface {
	Parse(*csv.Reader) ([]parsers.Trade, error)
}

// Presenter defines template data
type Presenter struct {
	LoggedIn  bool
	CSRFToken string
	Data      interface{}
	Form      interface{}
}

// FormField for persistent form values and messages in responses
type FormField struct {
	Value   string
	Message string
	Success bool
}

// Form to hold persistent posted-back values and a message
type Form struct {
	Fields  map[string]*FormField
	Message string
	Success bool
}

var (
	// SupportedCurrencies is a list of supported base currencies
	SupportedCurrencies = []string{
		"CAD",
		"USD",
	}
	// SupportedExchanges is a list of supported source exchanges
	SupportedExchanges = []string{
		"Coinbase",
		"Kucoin",
		"Cryptotax",
	}
	// TemplateFiles is a list of common template files needed for rendering
	TemplateFiles = []string{
		"web/templates/index.html.tmpl",
		"web/templates/hero.html.tmpl",
		"web/templates/navigation.html.tmpl",
		"web/templates/footer.html.tmpl",
	}
	// Config holds the app configuration
	Config *config.Configuration
)

func init() {
	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())
	// run the latest migrations
	database.Migrate()
	// load config values
	loadConfig()
}

// main: connect to database, set up router handlers and start web server.
func main() {
	// initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	// wrap DB
	env := &Env{db}

	// add router endpoints and handlers
	router := httprouter.New()
	router.GET("/", env.root)
	router.GET("/register", env.registration)
	router.POST("/register", env.register)
	router.GET("/verify_email", env.verifyEmail)
	router.GET("/login", env.loginPage)
	router.POST("/login", env.login)
	router.GET("/logout", env.logout)

	//router.POST("/newreport", env.newReport)
	//
	//router.GET("/currency", env.setCurrency)
	//router.POST("/currency", env.createReport)
	//
	//router.GET("/upload", env.manageFiles)
	//router.POST("/upload", env.uploadFile)
	//router.POST("/deletefile", env.deleteFile)
	//
	//router.GET("/trades", env.manageTrades)
	//router.POST("/trades", env.addTrade)
	//router.POST("/deletetrade", env.deleteTrade)
	//router.POST("/downloadtrades", env.downloadTrades)
	//
	//router.GET("/report", env.viewReport)

	// serve static files
	router.ServeFiles("/web/js/*filepath", http.Dir("web/js"))
	router.ServeFiles("/web/components/*filepath", http.Dir("web/components"))
	router.ServeFiles("/web/css/*filepath", http.Dir("web/css"))

	// start the web server
	log.Println("Web server ready.")
	log.Fatal(http.ListenAndServe(":5000", router))
}

// loadConfig reads the config file into the global Config variable
func loadConfig() {
	c, err := config.Read("config.json")
	if err != nil {
		log.Fatal(err)
	}
	Config = c
}

// initDB initializes the DB connection and returns a new instance
func initDB() (*models.DB, error) {
	d := Config.Database
	s := fmt.Sprintf("host=%v port=%v user=%v dbname=%v password=%v sslmode=%v",
		d.Host,
		d.Port,
		d.User,
		d.DBName,
		d.Password,
		d.SSLMode,
	)
	db, err := database.NewDB(s)
	if err != nil {
		log.Fatal(err)
	}
	db.LogMode(true)

	return db, nil
}

/*
func (env *Env) newReport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s, err := env.getSession(r)
	if err == nil && s != nil {
		env.db.KillSession(s.SessionID)
	}

	http.Redirect(w, r, "/currency", http.StatusSeeOther)
}

func (env *Env) setCurrency(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	t, err := template.ParseFiles(
		"templates/layout/base.tmpl",
		"templates/header.tmpl",
		"templates/layout/style.tmpl",
		"templates/layout/js.tmpl",
		"templates/currency.tmpl",
	)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	t.Execute(w, SupportedCurrencies)
}

func (env *Env) createReport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form parameters", http.StatusBadRequest)
		return
	}

	c := r.FormValue("currency")
	if !contains(SupportedCurrencies, c) {
		http.Error(w, "Invalid currency", http.StatusBadRequest)
		return
	}

	// retrieve and update report if there is an active session,
	// otherwise create it
	report := &models.Report{}
	s, err := env.getSession(r)
	if err == nil && s != nil {
		report, err = env.db.GetReport(s.ReportID)
		if err != nil || report == nil {
			http.Error(w, "Error fetching report", http.StatusInternalServerError)
			return
		}
		if c != report.Currency {
			report.Currency = c
			if err = env.db.UpdateReportCurrency(report); err != nil {
				http.Error(w, "Error updating report currency", http.StatusInternalServerError)
				return
			}
		}
	} else {
		report.Currency = c
		_, err := env.db.CreateReport(report)
		if err != nil {
			http.Error(w, "Error creating report", http.StatusInternalServerError)
			return
		}
		if err := env.setSessionCookie(w, report); err != nil {
			http.Error(w, "Couldn't set cookie", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/upload", http.StatusSeeOther)
}

func (env *Env) manageFiles(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// require an active session from this page on
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	fs, err := env.db.GetReportFiles(s.ReportID)
	if err != nil {
		http.Error(w, "Unable to retrieve files", http.StatusInternalServerError)
		return
	}

	t, err := template.ParseFiles(
		"templates/layout/base.tmpl",
		"templates/header.tmpl",
		"templates/layout/style.tmpl",
		"templates/layout/js.tmpl",
		"templates/upload.tmpl",
	)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	type Presenter struct {
		Exchanges []string
		Files     []*models.File
	}

	pr := &Presenter{
		Exchanges: SupportedExchanges,
		Files:     fs,
	}

	t.Execute(w, pr)
}

func (env *Env) uploadFile(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// requires active session
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// parse the form fields
	r.ParseMultipartForm(32 << 20)
	src := r.FormValue("exchange")
	if !contains(SupportedExchanges, src) {
		http.Error(w, "Invalid exchange", http.StatusBadRequest)
		return
	}

	// get the file
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		http.Error(w, "Bad file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// validate content type (can be faked by client)
	if !contains(handler.Header["Content-Type"], "text/csv") {
		http.Error(w, "Must be a CSV file", http.StatusInternalServerError)
		return
	}

	// calculate hash of file to prevent duplicates
	h := md5.New()
	if _, err = io.Copy(h, file); err != nil {
		http.Error(w, "Unable to get hash of file", http.StatusInternalServerError)
		return
	}
	hash := h.Sum(nil)
	file.Seek(0, 0) // reset file read pointer

	// parse the file into Trade records
	var ts []parsers.Trade
	switch src {
	case "Coinbase":
		ts, err = parse(&parsers.Coinbase{}, csv.NewReader(file))
	case "Kucoin":
		ts, err = parse(&parsers.Kucoin{}, csv.NewReader(file))
	case "Cryptotax":
		ts, err = parse(&parsers.Custom{}, csv.NewReader(file))
	default:
		http.Error(w, "Invalid exchange:\n"+err.Error(), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to parse CSV file: %v\n", err), http.StatusBadRequest)
		return
	}

	if len(ts) == 0 {
		http.Error(w, "No trades to process", http.StatusOK)
		return
	}

	// transaction for db inserts
	tx := env.db.BeginTransaction()

	if tx.Error != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}

	// store the File
	fs := &models.File{
		Name:     template.HTMLEscapeString(handler.Filename),
		Source:   src,
		Hash:     hash,
		ReportID: s.ReportID,
	}
	fid, err := tx.SaveFile(fs)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to store file hash", http.StatusInternalServerError)
		return
	}

	// store the Trades
	for _, t := range ts {
		trade := &models.Trade{
			Date:         t.Date,
			Asset:        t.Asset,
			Action:       t.Action,
			Quantity:     t.Quantity,
			BaseCurrency: t.BaseCurrency,
			BasePrice:    t.BasePrice,
			BaseFee:      t.BaseFee,
			FileID:       fid,
			ReportID:     s.ReportID,
		}
		_, err := tx.SaveTrade(trade)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Unable to save trades", http.StatusInternalServerError)
			return
		}
	}
	tx.Commit()

	http.Redirect(w, r, "/upload", http.StatusSeeOther)
}

func (env *Env) deleteFile(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// requires active session
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get hidden field's file ID
	fid := r.FormValue("fileid")
	fileID, err := strconv.ParseUint(fid, 0, 64)
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}
	// retrieve file to ensure session's report matches file report
	file, err := env.db.GetFile(uint(fileID))
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}
	if s.ReportID != file.ReportID {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	// do the delete
	if err := env.db.DeleteFile(file.ID); err != nil {
		http.Error(w, "Couldn't delete file", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/upload", http.StatusSeeOther)
}

func (env *Env) viewReport(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// requires active session
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get the report from session
	report, err := env.db.GetReport(s.ReportID)
	if err != nil {
		http.Error(w, "Unable to retrieve report", http.StatusInternalServerError)
		return
	}

	c, err := acb.Calculate(report.Trades, report.Currency)
	if err != nil {
		switch err.(type) {
		case *acb.Oversold:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Failed to calculate ACB", http.StatusInternalServerError)
		}
		return
	}

	sells, err := acb.SellOnly(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fm := template.FuncMap{
		"fiat": func(num decimal.Decimal) string {
			f, _ := num.Float64()
			return fmt.Sprintf("$%2.2f", f)
		},
	}

	files := []string{
		"templates/layout/base.tmpl",
		"templates/header.tmpl",
		"templates/layout/style.tmpl",
		"templates/layout/js.tmpl",
		"templates/report.tmpl",
	}

	t, err := template.New(path.Base(files[0])).Funcs(fm).ParseFiles(files...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Presenter struct {
		Report *models.Report
		ACBs   []*acb.ACB
	}

	pr := &Presenter{
		Report: report,
		ACBs:   sells,
	}

	t.Execute(w, pr)
}

func (env *Env) manageTrades(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// requires active session
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get manual trades
	ts, err := env.db.GetManualTrades(s.ReportID)
	if err != nil {
		http.Error(w, "Unable to fetch trades", http.StatusInternalServerError)
		return
	}

	t, err := template.ParseFiles(
		"templates/layout/base.tmpl",
		"templates/header.tmpl",
		"templates/layout/style.tmpl",
		"templates/layout/js.tmpl",
		"templates/trades.tmpl",
	)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	t.Execute(w, ts)
}

func (env *Env) addTrade(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// requires active session
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// build a Trade from the form
	if err = r.ParseForm(); err != nil {
		http.Error(w, "Invalid form parameters", http.StatusBadRequest)
		return
	}

	var date time.Time
	if date, err = time.Parse("2006-01-02", r.FormValue("date")); err != nil {
		http.Error(w, "Invalid date", http.StatusBadRequest)
		return
	}

	asset := strings.ToUpper(r.FormValue("asset"))
	if asset != html.EscapeString(asset) {
		http.Error(w, "Invalid asset", http.StatusBadRequest)
		return
	}

	action := strings.ToLower(r.FormValue("action"))
	if !(action == "buy" || action == "sell") {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	var quant decimal.Decimal
	if quant, err = decimal.NewFromString(r.FormValue("quantity")); err != nil {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	base := strings.ToUpper(r.FormValue("base"))
	if base != html.EscapeString(base) {
		http.Error(w, "Invalid base currency", http.StatusBadRequest)
		return
	}

	var cost decimal.Decimal
	if cost, err = decimal.NewFromString(r.FormValue("cost")); err != nil {
		http.Error(w, "Invalid cost", http.StatusBadRequest)
		return
	}

	var fee decimal.Decimal
	if fee, err = decimal.NewFromString(r.FormValue("fee")); err != nil {
		http.Error(w, "Invalid fee", http.StatusBadRequest)
		return
	}

	trade := &models.Trade{
		Date:         date,
		Asset:        asset,
		Action:       action,
		Quantity:     quant,
		BaseCurrency: base,
		BasePrice:    cost,
		BaseFee:      fee,
		ReportID:     s.ReportID,
	}

	// save the trade
	if id, err := env.db.SaveTrade(trade); err != nil {
		http.Error(w, fmt.Sprintf("Unable to save trade:\n%v\n%v\n", id, err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/trades", http.StatusSeeOther)
}

func (env *Env) deleteTrade(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// requires active session
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get the hidden field's trade ID
	tid := r.FormValue("tradeid")
	tradeID, err := strconv.ParseUint(tid, 0, 64)
	if err != nil {
		http.Error(w, "Invalid trade ID", http.StatusBadRequest)
		return
	}
	// retrieve trade to ensure session's report matches trade report
	trade, err := env.db.GetTrade(uint(tradeID))
	if err != nil {
		http.Error(w, "Invalid trade ID", http.StatusBadRequest)
		return
	}
	if s.ReportID != trade.ReportID {
		http.Error(w, "Invalid trade ID", http.StatusBadRequest)
		return
	}

	if err := env.db.DeleteTrade(trade.ID); err != nil {
		http.Error(w, "Couldn't delete trade", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/trades", http.StatusSeeOther)
}

func (env *Env) downloadTrades(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// requires active session
	s, err := env.getSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get manual trades
	ts, err := env.db.GetManualTrades(s.ReportID)
	if err != nil {
		http.Error(w, "Unable to fetch trades", http.StatusInternalServerError)
		return
	}

	c := &parsers.Custom{}
	csv, err := c.Generate(ts)
	if err != nil {
		http.Error(w, "Error generating CSV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=cryptotax-trades.csv")
	w.Header().Set("Content-Type", "text/csv")
	w.Write(csv)
}
*/

func parse(p Parser, r *csv.Reader) ([]parsers.Trade, error) {
	return p.Parse(r)
}
