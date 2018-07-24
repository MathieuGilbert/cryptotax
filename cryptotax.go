package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"html/template"
	"log"
	"net/http"
	"path"
	"runtime"
	"strings"

	"github.com/go-mail/mail"
	"github.com/goware/emailx"
	_ "github.com/jinzhu/gorm/dialects/postgres" // db driver
	"github.com/julienschmidt/httprouter"
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
)

func init() {
	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())
	// run the latest migrations
	database.Migrate()
}

// main: loads config file?, registers services, loads middleware for router,
// and sets up HTTP listener.
func main() {
	db, err := database.NewDB("host=localhost port=5432 user=cryptotax dbname=cryptotax_dev password=password!@# sslmode=disable")
	if err != nil {
		log.Panic(err)
	}
	db.LogMode(true)
	env := &Env{db}

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

	router.ServeFiles("/web/js/*filepath", http.Dir("web/js"))
	router.ServeFiles("/web/components/*filepath", http.Dir("web/components"))
	router.ServeFiles("/web/css/*filepath", http.Dir("web/css"))

	log.Println("Web server ready.")
	log.Fatal(http.ListenAndServe(":5000", router))
}

func (env *Env) root(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// check for existing session user
	u, _ := env.currentUser(r)
	if u == nil {
		// no session user, make new session
		if _, err := env.setSessionCookie(w, nil); err != nil {
			log.Printf("%+v", err)
			http.Error(w, "Unable to set cookie", http.StatusInternalServerError)
			return
		}
	}

	// build page
	t := template.Must(template.ParseFiles(append(TemplateFiles, "web/templates/root.html.tmpl")...))

	// define template data
	type Data struct {
		Name string
	}

	pr := &Presenter{
		LoggedIn: u != nil,
		Data:     &Data{Name: "teddy"},
	}

	t.Execute(w, pr)
}

func (env *Env) registration(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// check for existing session
	s, _ := env.getSession(r)
	if s == nil {
		// make new session
		ns, err := env.setSessionCookie(w, nil)
		if err != nil {
			log.Printf("%+v", err)
			http.Error(w, "Unable to set cookie", http.StatusInternalServerError)
			return
		}
		s.SessionID = ns.SessionID
	}

	// must not be logged in
	if s.UserID != 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// build page
	t, err := pageTemplate("web/templates/register.html.tmpl")
	if err != nil {
		log.Printf("%+v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	// define template data
	pr := &Presenter{
		LoggedIn:  false,
		CSRFToken: s.CSRFToken,
		Form:      &Form{},
	}

	t.Execute(w, pr)
}

func (env *Env) register(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// page redering details
	var file string
	pr := &Presenter{LoggedIn: false}

	// verify existing session
	s, _ := env.getSession(r)
	if s == nil {
		http.Error(w, "Session expired", http.StatusBadRequest)
		return
	}
	// must not be logged in
	if s.UserID != 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get form fields
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form parameters", http.StatusBadRequest)
		return
	}
	// verify CSRF token
	if r.FormValue("csrf_token") != s.CSRFToken {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}
	// read form
	// email: trim spaces, to lower case, html escape
	e := template.HTMLEscapeString(strings.TrimSpace(strings.ToLower(r.FormValue("email"))))
	p := r.FormValue("password")
	pp := r.FormValue("password_confirm")

	// build Form object for validation
	f := &Form{Fields: make(map[string]*FormField), Success: true}
	// persist fields
	f.Fields["email"] = &FormField{Value: e, Success: true}

	// validate inputs
	if err := emailx.Validate(e); err != nil {
		f.fail("email", "Invalid email.")
	}
	if len(p) < 8 {
		f.fail("password", "Password must be at least 8 characters.")
	}
	if p != pp {
		f.fail("password_confirm", "Passwords must match.")
	}
	if env.db.EmailExists(e) {
		f.fail("email", "Already registered. Use a different email or try logging in.")
	}

	if f.Success {
		// create user in DB
		u, err := env.db.RegisterUser(e, p)
		if err != nil {
			log.Printf("RegisterUser failed: %v\n", err)
			http.Error(w, "RegisterUser error.", http.StatusInternalServerError)
		}

		// build verification email template
		t := template.Must(template.ParseFiles(
			"web/templates/email/base.html.tmpl",
			"web/templates/email/verify_registration.html.tmpl",
		))
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, struct{ Token string }{Token: u.ConfirmToken}); err != nil {
			log.Printf("Error executing RegisterUser email template: %v\n", err)
			http.Error(w, "RegisterUser template error.", http.StatusInternalServerError)
		}

		// build email
		m := mail.NewMessage()
		m.SetHeader("From", "cryptotax@example.com")
		m.SetHeader("To", u.Email)
		m.SetHeader("Subject", "Cryptotax registration verification")
		m.SetBody("text/html", buf.String())

		// send email
		// using MailSlurper locally
		d := mail.NewDialer("localhost", 2500, "user", "pass")
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		if err := d.DialAndSend(m); err != nil {
			log.Printf("DialAndSend failed: %v\n", err)
			http.Error(w, "Error sending email.", http.StatusInternalServerError)
		}
	}

	if f.Success {
		file = "web/templates/register_success.html.tmpl"
		pr.Data = struct{ Email string }{Email: e}
	} else {
		f.Message = "Please fix the above errors to register."
		file = "web/templates/register.html.tmpl"
		pr.CSRFToken = s.CSRFToken
		pr.Form = f
	}

	t, err := pageTemplate(file)
	if err != nil {
		log.Printf("%+v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	t.Execute(w, pr)
}

func (env *Env) loginPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// redirect if already logged in
	s, _ := env.getSession(r)
	if s == nil {
		// make new session
		ns, err := env.setSessionCookie(w, nil)
		if err != nil {
			log.Printf("%+v", err)
			http.Error(w, "Unable to set cookie", http.StatusInternalServerError)
			return
		}
		s = ns
	}

	// must not be logged in
	if s.UserID != 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// build page
	t, err := pageTemplate("web/templates/login.html.tmpl")
	if err != nil {
		log.Printf("%+v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	// define template data
	pr := &Presenter{
		LoggedIn:  false,
		CSRFToken: s.CSRFToken,
	}

	t.Execute(w, pr)
}

func (env *Env) login(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// verify existing session
	s, _ := env.getSession(r)
	if s == nil {
		http.Error(w, "Session expired", http.StatusBadRequest)
		return
	}
	// must not be logged in
	if s.UserID != 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get form fields
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form parameters", http.StatusBadRequest)
		return
	}
	// verify CSRF token
	if r.FormValue("csrf_token") != s.CSRFToken {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}
	// read form
	// email: trim spaces, to lower case, html escape
	e := template.HTMLEscapeString(strings.TrimSpace(strings.ToLower(r.FormValue("email"))))
	p := r.FormValue("password")

	// build Form object for validation
	f := &Form{Success: true, Message: ""}

	// try logging in
	u, err := env.db.Authenticate(e, p)

	// failed login
	if err != nil {
		f.Success = false
		f.Message = "Invalid credentials."
	}
	// user has not verified email
	if !u.Confirmed {
		f.Success = false
		f.Message = "Please check your email for the verification link."
	}

	// login failure
	if !f.Success {
		pr := &Presenter{
			LoggedIn:  false,
			CSRFToken: s.CSRFToken,
			Form:      f,
		}

		// build page
		t, err := pageTemplate("web/templates/login.html.tmpl")
		if err != nil {
			log.Printf("%+v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}

		t.Execute(w, pr)
		return
	}

	// successful login
	if err := env.db.UpgradeSession(s, u); err != nil {
		http.Error(w, "Session error logging in", http.StatusInternalServerError)
		return
	}

	// logged in, return to root
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (env *Env) logout(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s, _ := env.getSession(r)

	if s != nil && s.UserID != 0 {
		env.db.KillSession(s)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (f *Form) fail(field, message string) {
	f.Fields[field].Message = message
	f.Fields[field].Success = false
	f.Success = false
}

func pageTemplate(t string) (*template.Template, error) {
	ts := append(TemplateFiles, t)
	return template.New(path.Base(ts[0])).Funcs(funcMaps()).ParseFiles(ts...)
}

func funcMaps() map[string]interface{} {
	return template.FuncMap{
		"hasMessage":   hasMessage,
		"fieldMessage": fieldMessage,
		"fieldClass":   fieldClass,
		"fieldValue":   fieldValue,
	}
}

func hasMessage(field string, form *Form) bool {
	if f := form.Fields[field]; f != nil {
		return f.Message != ""
	}
	return false
}

func fieldClass(field string, form *Form) string {
	if f := form.Fields[field]; f != nil {
		if f.Success {
			return "success"
		}
		return "danger"
	}
	return ""
}

func fieldMessage(field string, form *Form) string {
	if f := form.Fields[field]; f != nil {
		return f.Message
	}
	return ""
}

func fieldValue(field string, form *Form) string {
	if f := form.Fields[field]; f != nil {
		return f.Value
	}
	return ""
}

func (env *Env) verifyEmail(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// build page
	t, err := pageTemplate("web/templates/verify_email.html.tmpl")
	if err != nil {
		log.Printf("%+v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	// define template data
	pr := &Presenter{
		LoggedIn: false,
	}

	// get token from query string
	token := r.URL.Query().Get("t")
	// attempt to verify the token
	ok := env.db.VerifyEmail(token)
	pr.Data = struct{ Confirmed bool }{Confirmed: ok}

	t.Execute(w, pr)
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

func contains(list []string, item string) bool {
	if item == "" {
		return false
	}

	for _, i := range list {
		if item == i {
			return true
		}
	}
	return false
}

func (env *Env) setSessionCookie(w http.ResponseWriter, u *models.User) (*models.Session, error) {
	s, err := env.db.NewSession(u)
	if err != nil {
		return nil, err
	}
	cookie := &http.Cookie{
		Name:     "cryptotax",
		Value:    s.SessionID,
		HttpOnly: true,
		Path:     "/",
		Secure:   false, // TODO: use config to secure on live servers
		Expires:  s.Expires,
	}
	http.SetCookie(w, cookie)
	return s, nil
}

// getSession reads sessionID from cookie and return that session from the database
func (env *Env) getSession(r *http.Request) (*models.Session, error) {
	cookies := r.Cookies()
	cValue := ""
	for _, c := range cookies {
		if c.Name == "cryptotax" && c.Value != "" {
			cValue = c.Value
		}
	}
	if cValue == "" {
		return nil, nil
	}
	s, err := env.db.GetSession(cValue)
	if err != nil {
		return nil, nil
	}
	return s, err
}

func (env *Env) currentUser(r *http.Request) (*models.User, error) {
	s, err := env.getSession(r)
	if err != nil {
		return nil, err
	}
	if s == nil || s.UserID == 0 {
		return nil, nil
	}

	u, err := env.db.GetUser(s.UserID)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func parse(p Parser, r *csv.Reader) ([]parsers.Trade, error) {
	return p.Parse(r)
}
