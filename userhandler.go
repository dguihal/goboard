package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"
	goboardcookie "github.com/dguihal/goboard/cookie"
	goboarduser "github.com/dguihal/goboard/user"
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
}

func (u *UserHandler) AddUser(w http.ResponseWriter, login string, passwd string) {
	err := goboarduser.AddUser(u.db, login, passwd)

	if err == nil {
		// User created : Send him a cookie
		if cookie, err := goboardcookie.CookieForUser(u.db, login, u.cookieDuration_d); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
		} else {
			http.SetCookie(w, &cookie)
			w.WriteHeader(http.StatusOK)
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err.Error())
	}
}

func (u *UserHandler) AuthUser(w http.ResponseWriter, login string, passwd string) {

	if err := goboarduser.AuthUser(u.db, login, passwd); err != nil {
		fmt.Println("goboarduser.AuthUser : ", err.Error())
		if uerr, ok := err.(*goboarduser.UserError); ok {
			switch uerr.ErrCode {
			case goboarduser.AuthenticationFailed:
				w.WriteHeader(http.StatusUnauthorized)
			case goboarduser.DatabaseError:
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println(err.Error())
			}
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
		}
	} else {
		// User authenticated : Send him a cookie
		if cookie, err := goboardcookie.CookieForUser(u.db, login, u.cookieDuration_d); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
		} else {
			http.SetCookie(w, &cookie)
			w.WriteHeader(http.StatusOK)
		}
	}
}
