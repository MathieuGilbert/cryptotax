package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mail/mail"
	"github.com/goware/emailx"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
	"github.com/mathieugilbert/cryptotax/cmd/parsers"
	"github.com/mathieugilbert/cryptotax/models"
)

func (env *Env) getRoot(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
	}

	t.Execute(w, pr)
}

func (env *Env) getRegister(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// check for existing session
	s, _ := env.session(r)
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

func (env *Env) postRegister(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// page redering details
	var file string
	pr := &Presenter{LoggedIn: false}

	// verify existing session
	s, _ := env.session(r)
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

func (env *Env) getVerify(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// build page
	t, err := pageTemplate("web/templates/verify.html.tmpl")
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

func (env *Env) getLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// redirect if already logged in
	s, _ := env.session(r)
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

func (env *Env) postLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// verify existing session
	s, _ := env.session(r)
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

func (env *Env) getLogout(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s, _ := env.session(r)

	if s != nil && s.UserID != 0 {
		env.db.KillSession(s)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (env *Env) getFiles(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// require an active session and user
	u, err := env.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	fs, err := env.db.GetFiles(u.ID)
	if err != nil {
		log.Printf("Error getting user files: %v\n", err)
		http.Error(w, "Error retrieving files", http.StatusInternalServerError)
		return
	}

	// build page
	t, err := pageTemplate("web/templates/manage_files.html.tmpl")
	if err != nil {
		log.Printf("%+v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	pr := &Presenter{
		LoggedIn: true,
		Data: struct {
			Exchanges []string
			Files     []*models.File
		}{
			Exchanges: SupportedExchanges,
			Files:     fs,
		},
	}

	t.Execute(w, pr)
}

func (env *Env) postUploadAsync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// requires active session and user
	s, err := env.session(r)
	if err != nil || s.UserID == 0 {
		http.Error(w, "Expired session", http.StatusBadRequest)
		return
	}

	// posted JSON structure
	type Data struct {
		FileText  string
		FileBytes string
		Exchange  string
		FileName  string
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
		FileID   uint   `json:"file_id"`
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
	switch data.Exchange {
	case "Coinbase":
		ts, err = parse(&parsers.Coinbase{}, cr)
	case "Kucoin":
		ts, err = parse(&parsers.Kucoin{}, cr)
	case "Cryptotax":
		ts, err = parse(&parsers.Custom{}, cr)
	default:
		resp.Success = false
		resp.Message = fmt.Sprintf("File does not match %v format.", data.Exchange)
	}
	if err != nil {
		resp.Success = false
		resp.Message = "Unable to process exchange file."
	} else {
		if len(ts) == 0 {
			resp.Success = false
			resp.Message = "No trades found in file."
		}
	}

	// transaction for db inserts
	tx := env.db.BeginTransaction()

	if tx.Error != nil {
		log.Println("Error starting transaction")
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}

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

	if resp.Success {
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

func (env *Env) deleteFileAsync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// requires active session and user
	s, err := env.session(r)
	if err != nil || s.UserID == 0 {
		http.Error(w, "Expired session", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
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

func (env *Env) getFileTradesAsync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// requires active session and user
	s, err := env.session(r)
	if err != nil || s.UserID == 0 {
		http.Error(w, "Expired session", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	fid, err := strconv.ParseUint(q.Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid file id", http.StatusBadRequest)
		return
	}

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
