package main

import (
	"fmt"
	"net/http"

	"github.com/boltdb/bolt"
)

type UserHandler struct {
	GoboardHandler

	cookieDuration_d int
}

func NewUserHandler(db *bolt.DB, cookieDuration int) (u *UserHandler) {
	u = &UserHandler{}
	u.db = db

	u.supportedOps = []SupportedOp{
		{"/user/add", "POST"},   // Add a user
		{"/user/login", "POST"}, // Sign in a user
	}

	u.cookieDuration_d = cookieDuration

	return
}

func (u *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

/*
	switch r.Method {
	case "POST":
		fmt.Println("POST")
		r.ParseForm()

		loginAttr := r.FormValue("login")
		passwdAttr := r.FormValue("password")

		if strings.HasSuffix(r.URL.Path, "add") {
			u.AddUser(w, loginAttr, passwdAttr)
		} else if strings.HasSuffix(r.URL.Path, "login") {
			u.AuthUser(w, loginAttr, passwdAttr)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
} */

func (u *UserHandler) AddUser(w http.ResponseWriter, login string, passwd string) {
	cookie, err := AddUser(u.db, login, passwd)

	if err == nil {
		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (u *UserHandler) AuthUser(w http.ResponseWriter, login string, passwd string) {

	if cookie, err := AuthUser(u.db, login, passwd); err != nil {
		if uerr, ok := err.(*UserError); ok {
			switch uerr.ErrCode {
			case AuthenticationFailed:
				w.WriteHeader(http.StatusUnauthorized)
			case DatabaseError:
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Errorf("Internal error : %s", err.Error())
				return
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Errorf("Internal error : %s", err.Error())
		}
	} else {

		if cookie.Name == "" {
			// No Valid cookie found, create one
			cookie, err = CreateAndStoreCookie(u.db, login, u.cookieDuration_d)
		}

		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	}
}
