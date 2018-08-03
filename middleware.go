package main

import (
	"context"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Wrap httprouter's method signature, putting extra param into context.
// Access params with: ps, ok := r.Context().Value("params").(httprouter.Params)
// https://github.com/julienschmidt/httprouter/issues/198
func (env *Env) wrapHandler(h http.HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		type contextKey string
		// Take the context out from the request
		ctx := r.Context()

		// Get new context with key-value "params" -> "httprouter.Params"
		ctx = context.WithValue(ctx, contextKey("params"), ps)

		// Get new http.Request with the new context
		r = r.WithContext(ctx)

		// Call your original http.Handler
		h.ServeHTTP(w, r)
	}
}

func (env *Env) loggedInOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// requires active session and user
		s, err := env.session(r)
		if err != nil || s.UserID == 0 {
			http.Error(w, "Expired session", http.StatusBadRequest)
			return
		}
		h(w, r)
	}
}

func (env *Env) requireSession(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, _ := env.session(r)
		if s == nil {
			// make new session
			_, err := env.setSessionCookie(w, nil)
			if err != nil {
				log.Printf("%+v", err)
				http.Error(w, "Unable to set cookie", http.StatusInternalServerError)
				return
			}
		}
		h(w, r)
	}
}

func (env *Env) notLoggedIn(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, _ := env.session(r)
		if s != nil && s.UserID != 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		h(w, r)
	}
}
