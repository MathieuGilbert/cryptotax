package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mail/mail"
	"github.com/goware/emailx"
	"github.com/lib/pq"
	"github.com/mathieugilbert/cryptotax/cmd/parsers"
	"github.com/mathieugilbert/cryptotax/cmd/reports"
	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

func (env *Env) getRoot(w http.ResponseWriter, r *http.Request) {
	s, _ := env.session(r)

	pr := &Presenter{
		LoggedIn: s.UserID != 0,
	}

	t := pageTemplate("web/templates/root.html.tmpl")
	t.Execute(w, pr)
}

func (env *Env) getRegister(w http.ResponseWriter, r *http.Request) {
	s, _ := env.session(r)

	pr := &Presenter{
		LoggedIn:  false,
		CSRFToken: s.CSRFToken,
		Form:      &Form{},
	}

	t := pageTemplate("web/templates/register.html.tmpl")
	t.Execute(w, pr)
}

func (env *Env) postRegister(w http.ResponseWriter, r *http.Request) {
	// get form fields
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form parameters", http.StatusBadRequest)
		return
	}

	s, _ := env.session(r)

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
		m.SetHeader("From", "cryptotax@example.com") // TODO: from config
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

	var file string
	pr := &Presenter{LoggedIn: false}

	if f.Success {
		file = "web/templates/register_success.html.tmpl"
		pr.Data = struct{ Email string }{Email: e}
	} else {
		f.Message = "Please fix the above errors to register."
		file = "web/templates/register.html.tmpl"
		pr.CSRFToken = s.CSRFToken
		pr.Form = f
	}

	t := pageTemplate(file)
	t.Execute(w, pr)
}

func (env *Env) getVerify(w http.ResponseWriter, r *http.Request) {
	pr := &Presenter{
		LoggedIn: false,
	}

	// get token from query string
	token := r.URL.Query().Get("t")

	// attempt to verify the token
	ok := env.db.VerifyEmail(token)
	pr.Data = struct{ Confirmed bool }{Confirmed: ok}

	t := pageTemplate("web/templates/verify.html.tmpl")
	t.Execute(w, pr)
}

func (env *Env) getLogin(w http.ResponseWriter, r *http.Request) {
	s, _ := env.session(r)

	pr := &Presenter{
		LoggedIn:  false,
		CSRFToken: s.CSRFToken,
	}

	t := pageTemplate("web/templates/login.html.tmpl")
	t.Execute(w, pr)
}

func (env *Env) postLogin(w http.ResponseWriter, r *http.Request) {
	// get form fields
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form parameters", http.StatusBadRequest)
		return
	}

	s, _ := env.session(r)

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

		t := pageTemplate("web/templates/login.html.tmpl")
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

func (env *Env) getLogout(w http.ResponseWriter, r *http.Request) {
	s, _ := env.session(r)

	if s != nil && s.UserID != 0 {
		env.db.KillSession(s)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (env *Env) getFiles(w http.ResponseWriter, r *http.Request) {
	s, _ := env.session(r)

	fs, err := env.db.GetFiles(s.UserID)
	if err != nil {
		log.Printf("Error getting user files: %v\n", err)
		http.Error(w, "Error retrieving files", http.StatusInternalServerError)
		return
	}

	pr := &Presenter{
		LoggedIn:  true,
		CSRFToken: s.CSRFToken,
		Data: struct {
			Exchanges []string
			Files     []*models.File
		}{
			Exchanges: SupportedExchanges,
			Files:     fs,
		},
	}

	t := pageTemplate(
		"web/templates/manage_files.html.tmpl",
		"web/templates/components/file_manager.html.tmpl",
	)
	t.Execute(w, pr)
}

func (env *Env) postUploadAsync(w http.ResponseWriter, r *http.Request) {
	// posted JSON structure
	type Data struct {
		FileBytes string
		Exchange  string
		FileName  string
		CSRFToken string
	}
	// read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request", http.StatusInternalServerError)
		return
	}

	// unmarshal json body into Data
	var data Data
	if err = json.Unmarshal(body, &data); err != nil {
		log.Printf("unmarshal error: %v\n", err)
		http.Error(w, "Error during JSON unmarshal", http.StatusInternalServerError)
		return
	}

	// verify CSRF token
	if !env.validToken(r, data.CSRFToken) {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	// base64 decode FileBytes
	b, err := base64.StdEncoding.DecodeString(data.FileBytes)
	if err != nil {
		log.Printf("error decoding bytes: %v\n", err)
		http.Error(w, "Error decoding file bytes", http.StatusInternalServerError)
		return
	}

	// convert decoded bytes into a string
	bs := string(b)
	// split the string array: "[100,90,80]"
	ss := strings.Split(bs[1:len(bs)-1], ",")
	// convert to new byte slice
	var ba []byte
	for _, s := range ss {
		i, _ := strconv.Atoi(s)
		ba = append(ba, byte(i))
	}

	fileName := template.HTMLEscapeString(data.FileName)

	// struct for response data
	type Response struct {
		FileID   uint   `json:"fileId"`
		Name     string `json:"name"`
		Date     string `json:"date"`
		Exchange string `json:"exchange"`
		Message  string `json:"message"`
		Success  bool   `json:"success"`
	}
	resp := &Response{
		Name:    fileName,
		Success: true,
	}

	// parse the file into Trade records
	var ts []parsers.Trade
	cr := csv.NewReader(strings.NewReader(string(ba)))
	p, err := parsers.NewParser(data.Exchange)
	if err != nil {
		resp.Success = false
		resp.Message = fmt.Sprintf("File does not match %v format.", data.Exchange)
	}

	ts, err = parse(p.(Parser), cr)
	if err != nil {
		resp.Success = false
		resp.Message = "Unable to process exchange file."
	} else {
		if len(ts) == 0 {
			resp.Success = false
			resp.Message = "No trades found in file."
		}
	}

	if resp.Success {
		// transaction for db inserts
		tx := env.db.BeginTransaction()

		if tx.Error != nil {
			log.Println("Error starting transaction")
			http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
			return
		}

		s, _ := env.session(r)

		// store the File
		fs := &models.File{
			Name:   fileName,
			Source: data.Exchange,
			Bytes:  ba,
			UserID: s.UserID,
		}
		fid, err := tx.SaveFile(fs)
		if err != nil {
			tx.Rollback()

			if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
				resp.Success = false
				resp.Message = "File already exists."
			} else {
				log.Printf("Failed to save file: %v\n", fileName)
				http.Error(w, "Failed to save file", http.StatusInternalServerError)
				return
			}
		} else {
			resp.FileID = fid
		}

		// store the Trades
		for _, t := range ts {
			trade := &models.Trade{
				Date:         t.Date,
				Action:       t.Action,
				Amount:       t.Amount,
				Currency:     t.Currency,
				BaseAmount:   t.BaseAmount,
				BaseCurrency: t.BaseCurrency,
				FeeAmount:    t.FeeAmount,
				FeeCurrency:  t.FeeCurrency,
				FileID:       fid,
				UserID:       s.UserID,
			}
			_, err := tx.SaveTrade(trade)
			if err != nil {
				tx.Rollback()
				log.Printf("Error saving trade: %v, error: %v\n", trade, err)
				http.Error(w, "Unable to save trades", http.StatusInternalServerError)
				return
			}
		}
		tx.Commit()

		t := time.Now()
		resp.Date = fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d-00:00",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
		resp.Exchange = data.Exchange
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (env *Env) deleteFileAsync(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s, _ := env.session(r)

	// verify CSRF token
	if q.Get("csrf_token") != s.CSRFToken {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseUint(q.Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid file id", http.StatusBadRequest)
		return
	}

	if err = env.db.DeleteFile(uint(id), s.UserID); err != nil {
		http.Error(w, "Unable to delete file", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("")
}

func (env *Env) getFileTradesAsync(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fid, err := strconv.ParseUint(q.Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid file id", http.StatusBadRequest)
		return
	}

	s, _ := env.session(r)

	ts, err := env.db.GetFileTrades(uint(fid), s.UserID)
	if err != nil {
		http.Error(w, "Error getting trades", http.StatusBadRequest)
		return
	}

	type Response struct {
		Trades []*models.Trade `json:"trades"`
	}
	resp := &Response{Trades: ts}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (env *Env) getTrades(w http.ResponseWriter, r *http.Request) {
	s, _ := env.session(r)

	ts, err := env.db.GetManualTrades(s.UserID)
	if err != nil {
		log.Printf("Error getting user trades: %v\n", err)
		http.Error(w, "Error retrieving files", http.StatusInternalServerError)
		return
	}

	pr := &Presenter{
		LoggedIn:  true,
		CSRFToken: s.CSRFToken,
		Data: struct {
			Trades []*models.Trade
		}{
			Trades: ts,
		},
	}

	t := pageTemplate(
		"web/templates/components/trade_manager.html.tmpl",
		"web/templates/manage_trades.html.tmpl",
	)
	t.Execute(w, pr)
}

func (env *Env) postTradeAsync(w http.ResponseWriter, r *http.Request) {
	// posted JSON structure
	type Trade struct {
		Date         string
		Action       string
		Amount       string
		Currency     string
		BaseAmount   string
		BaseCurrency string
		FeeAmount    string
		FeeCurrency  string
	}
	type Data struct {
		Trade     *Trade
		CSRFToken string
	}
	// read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request", http.StatusInternalServerError)
		return
	}

	// unmarshal json body into Data
	var data Data
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("unmarshal error: %v\n", err)
		http.Error(w, "Error during JSON unmarshal", http.StatusInternalServerError)
		return
	}

	s, _ := env.session(r)

	// verify CSRF token
	if data.CSRFToken != s.CSRFToken {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	// validate trade
	var date time.Time
	if date, err = time.Parse("2006-01-02", data.Trade.Date); err != nil {
		http.Error(w, "Invalid date.", http.StatusBadRequest)
		return
	}

	action := strings.ToUpper(data.Trade.Action)
	// skip if not a buy or sell
	if action != "BUY" && action != "SELL" {
		http.Error(w, "Must be BUY or SELL.", http.StatusBadRequest)
		return
	}

	var amount decimal.Decimal
	if amount, err = decimal.NewFromString(data.Trade.Amount); err != nil {
		http.Error(w, "Invalid amount.", http.StatusBadRequest)
		return
	}

	currency := strings.ToUpper(html.EscapeString(data.Trade.Currency))

	var baseAmount decimal.Decimal
	if baseAmount, err = decimal.NewFromString(data.Trade.BaseAmount); err != nil {
		http.Error(w, "Invalid base amount.", http.StatusBadRequest)
		return
	}

	baseCurrency := strings.ToUpper(html.EscapeString(data.Trade.BaseCurrency))

	var feeAmount decimal.Decimal
	if feeAmount, err = decimal.NewFromString(data.Trade.FeeAmount); err != nil {
		http.Error(w, "Invalid base amount.", http.StatusBadRequest)
		return
	}

	feeCurrency := strings.ToUpper(html.EscapeString(data.Trade.FeeCurrency))

	trd := &models.Trade{
		Date:         date,
		Action:       action,
		Amount:       amount,
		Currency:     currency,
		BaseAmount:   baseAmount,
		BaseCurrency: baseCurrency,
		FeeAmount:    feeAmount,
		FeeCurrency:  feeCurrency,
		UserID:       s.UserID,
	}
	t, err := env.db.SaveTrade(trd)
	if err != nil {
		log.Printf("Error saving trade: %v\n%v\n", trd, err)
		http.Error(w, "Error saving trade.", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Trade *models.Trade `json:"trade"`
	}
	resp := &Response{Trade: t}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (env *Env) deleteTradeAsync(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s, _ := env.session(r)

	// verify CSRF token
	if q.Get("csrf_token") != s.CSRFToken {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	// get query params
	id, err := strconv.ParseUint(q.Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid trade id", http.StatusBadRequest)
		return
	}

	if err = env.db.DeleteTrade(uint(id), s.UserID); err != nil {
		http.Error(w, "Unable to delete trade", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("")
}

func (env *Env) getReports(w http.ResponseWriter, r *http.Request) {
	s, _ := env.session(r)

	pr := &Presenter{
		LoggedIn:  true,
		CSRFToken: s.CSRFToken,
	}

	t := pageTemplate(
		"web/templates/reports.html.tmpl",
		"web/templates/components/report_viewer.html.tmpl",
	)
	t.Execute(w, pr)
}

func (env *Env) getReportAsync(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s, _ := env.session(r)

	// verify CSRF token
	if q.Get("csrf_token") != s.CSRFToken {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	// get query params
	t := q.Get("type")
	if t != "Holdings" && t != "ACB" {
		http.Error(w, "Invalid report type", http.StatusBadRequest)
		return
	}

	c := q.Get("currency")
	if c != "CAD" {
		http.Error(w, "Invalid currency", http.StatusBadRequest)
		return
	}

	a := q.Get("asof")
	if a != "Today" && a != "EOY2017" {
		http.Error(w, "Invalid as of", http.StatusBadRequest)
		return
	}

	rpt := &reports.Holdings{Currency: c}

	ts, err := env.db.GetUserTrades(s.UserID)
	if err != nil {
		http.Error(w, "Error getting user trades", http.StatusInternalServerError)
		return
	}

	type Item struct {
		Asset  string          `json:"asset"`
		Amount decimal.Decimal `json:"amount"`
		ACB    decimal.Decimal `json:"acb"`
		Value  decimal.Decimal `json:"value"`
		Gain   decimal.Decimal `json:"gain"`
	}
	type Response struct {
		Items []*Item `json:"items"`
		Error string  `json:"error"`
	}
	resp := &Response{}

	err = rpt.Build(ts, convert)
	if err != nil {
		switch err.(type) {
		case *reports.Oversold:
			resp.Error = err.Error()
		default:
			http.Error(w, "Error building report", http.StatusInternalServerError)
			return
		}
	}

	// add the items
	for _, i := range rpt.Items {
		resp.Items = append(resp.Items, &Item{
			Asset:  i.Asset,
			Amount: i.Amount,
			ACB:    i.ACB,
			Value:  i.Value,
			Gain:   i.Gain,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
