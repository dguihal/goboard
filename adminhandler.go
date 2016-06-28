package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"
	goboardcookie "github.com/dguihal/goboard/cookie"
	goboarduser "github.com/dguihal/goboard/user"
)

type AdminHandler struct {
	GoboardHandler

	adminToken string
}

func NewAdminHandler(db *bolt.DB) (a *AdminHandler) {
	a = &AdminHandler{}

	a.db = db

	a.supportedOps = []SupportedOp{
		{"/admin/user", "DELETE"}, // Delete a user
		{"/admin/post", "DELETE"}, // Delete a post
	}

	a.adminToken = "plop"
	return
}

func (a *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqAdminToken := r.Header.Get("Token-Id")
	if !a.checkAdminToken(reqAdminToken) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "DELETE":
		if strings.HasSuffix(r.URL.Path, "user") {
			loginAttr := r.FormValue("login")
			a.DeleteUser(w, loginAttr)
		}
	}
}

func (a *AdminHandler) DeleteUser(w http.ResponseWriter, login string) {
	if err := goboarduser.DeleteUser(a.db, login); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}

	if err := goboardcookie.DeleteCookiesForUser(a.db, login); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	return

}

func (a *AdminHandler) checkAdminToken(token string) bool {
	return token == a.adminToken
}
