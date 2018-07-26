package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-mail/mail"
	"github.com/goware/emailx"
	"github.com/julienschmidt/httprouter"
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
		Data:     &Data{Name: "teddy"},
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
	_, err := env.currentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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
		Data: struct{ Exchanges []string }{SupportedExchanges},
		//Files:     fs,
	}

	t.Execute(w, pr)
}

func (env *Env) postUploadAsync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// requires active session
	_, err := env.session(r)
	if err != nil {
		http.Error(w, "Expired session", http.StatusBadRequest)
		return
	}

	// parse the form fields
	r.ParseMultipartForm(32 << 20)

	// struct for response data
	type File struct {
		Hash    string `json:"hash"`
		Name    string `json:"name"`
		Date    string `json:"date"`
		Message string `json:"message"`
		Success bool   `json:"success"`
	}
	type Response struct {
		Files []File `json:"files"`
	}
	resp := &Response{}

	m := r.MultipartForm
	fhs := m.File["file"]
	for _, fh := range fhs {
		success := true
		var msg string
		//for each fileheader, get a handle to the actual file
		file, err := fh.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		//src := r.FormValue("exchange")
		//if !contains(SupportedExchanges, src) {
		//	http.Error(w, "Invalid exchange", http.StatusBadRequest)
		//	return
		//}

		// validate content type (can be faked by client)
		if !contains(fh.Header["Content-Type"], "text/csv") {
			success = false
			msg = "Invalid CSV file."
		}

		// calculate hash of file to prevent duplicates
		h := md5.New()
		if _, err = io.Copy(h, file); err != nil {
			log.Printf("Error getting hash of file")
			http.Error(w, "Unable to get hash of file", http.StatusInternalServerError)
			return
		}
		hash := h.Sum(nil)
		file.Seek(0, 0) // reset file read pointer

		var d string
		if success {
			t := time.Now()
			d = fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d-00:00",
				t.Year(), t.Month(), t.Day(),
				t.Hour(), t.Minute(), t.Second())
		}

		resp.Files = append(resp.Files, File{
			Hash:    string(hash),
			Name:    fh.Filename,
			Date:    d,
			Message: msg,
			Success: success,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

/*
func (env *Env) postUpload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {


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
*/
