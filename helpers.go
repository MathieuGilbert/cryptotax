package main

import (
	"html/template"
	"net/http"
	"path"
	"time"

	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

func (f *Form) fail(field, message string) {
	f.Fields[field].Message = message
	f.Fields[field].Success = false
	f.Success = false
}

func pageTemplate(ts ...string) (*template.Template, error) {
	f := TemplateFiles
	for _, t := range ts {
		f = append(f, t)
	}
	return template.New(path.Base(f[0])).Funcs(funcMaps()).ParseFiles(f...)
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

// session reads sessionID from cookie and return that session from the database
func (env *Env) session(r *http.Request) (*models.Session, error) {
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
	s, err := env.db.Session(cValue)
	if err != nil {
		return nil, nil
	}
	return s, err
}

func (env *Env) currentUser(r *http.Request) (*models.User, error) {
	s, err := env.session(r)
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

func convert(amount decimal.Decimal, from, to string, on time.Time) (decimal.Decimal, error) {
	if from == to {
		return amount, nil
	}

	return amount.Mul(decimal.NewFromFloat(2)), nil
}
