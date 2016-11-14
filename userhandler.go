package main

import (
	"encoding/json"
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
		{"/user/add", "/user/add", "POST", u.addUser},      // Add a user
		{"/user/login", "/user/login", "POST", u.authUser}, // Authenticate a user
		{"/user/whoami", "/user/whoami", "GET", u.whoAmI},  // Get self account infos
	}

	u.cookieDuration_d = cookieDuration

	return
}

func (u *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	for _, op := range u.supportedOps {
		if r.Method == op.Method && strings.HasPrefix(r.URL.Path, op.PathBase) {
			// Call specific handling method
			op.handler(w, r)
			return
		}
	}
}

func (u *UserHandler) addUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var login, passwd string = "", ""
	if login = r.FormValue("login"); len(login) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Login can't be empty"))
		return
	}

	if passwd := r.FormValue("password"); len(passwd) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Password can't be empty"))
	}

	err := goboarduser.AddUser(u.db, login, passwd)

	if err == nil {
		// User created : Send him a cookie
		if cookie, err := goboardcookie.CookieForUser(u.db, login, u.cookieDuration_d); err == nil {
			http.SetCookie(w, &cookie)
			w.WriteHeader(http.StatusOK)
			return
		}
	} else {
		if uerr, ok := err.(*goboarduser.UserError); ok {
			if uerr.ErrCode == goboarduser.UserAlreadyExistsError {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte("User login already exists"))
				return
			}
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Println(err.Error())

	return
}

func (u *UserHandler) authUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var login, passwd string = "", ""
	if login = r.FormValue("login"); len(login) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Login can't be empty"))
		return
	}

	if passwd := r.FormValue("password"); len(passwd) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Password can't be empty"))
	}

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

func (u *UserHandler) whoAmI(w http.ResponseWriter, r *http.Request) {

	if cookies := r.Cookies(); len(cookies) > 0 {
		if login, err := goboardcookie.LoginForCookie(u.db, cookies[0].Value); err == nil && len(login) > 0 {
			var err error
			var user goboarduser.User

			if user, err = goboarduser.GetUser(u.db, login); err == nil {
				var data []byte

				if data, err = json.Marshal(user); err == nil {
					w.WriteHeader(http.StatusOK)
					w.Write(data)
					return
				}
			}

			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
			return
		}
	}

	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("You need to be authenticated"))
	return
}
