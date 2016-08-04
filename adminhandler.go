package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	goboardbackend "github.com/dguihal/goboard/backend"
	goboardcookie "github.com/dguihal/goboard/cookie"
	goboarduser "github.com/dguihal/goboard/user"
	"github.com/gorilla/mux"
)

const tokenMinLen int = 0
const tokenWarnLen int = 12

type AdminHandler struct {
	GoboardHandler

	adminToken string
}

func NewAdminHandler(db *bolt.DB, adminToken string) (a *AdminHandler) {
	a = &AdminHandler{}

	a.db = db

	a.supportedOps = []SupportedOp{
		{"/admin/user/", "/admin/user/{login}", "DELETE", a.deleteUser}, // Delete a user
		{"/admin/post/", "/admin/post/{id}", "DELETE", a.deletePost},    // Delete a post
	}

	if len(adminToken) <= tokenMinLen {
		log.Println("Admin token empty : for security reasongs, this means that no admin operations will be authorized")
	} else if len(adminToken) < tokenWarnLen {
		log.Println("Admin token len <", tokenWarnLen, ": Come on I'm sure you can do a lot better")
	}
	a.adminToken = adminToken
	return
}

func (a *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	reqAdminToken := r.Header.Get("Token-Id")
	if !a.checkAdminToken(reqAdminToken) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	for _, op := range a.supportedOps {
		if r.Method == op.Method && strings.HasPrefix(r.URL.Path, op.PathBase) {
			// Call specific handling method
			op.handler(w, r)
			return
		}
	}

	// If we are here : not methods has been found (shouldn't happen)
	w.WriteHeader(http.StatusNotFound)
	return
}

func (a *AdminHandler) deleteUser(w http.ResponseWriter, r *http.Request) {

	login := (mux.Vars(r))["login"]

	if err := goboarduser.DeleteUser(a.db, login); err != nil {
		if uerr, ok := err.(*goboarduser.UserError); ok {
			if uerr.ErrCode == goboarduser.UserDoesNotExistsError {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(fmt.Sprintf("User %s Not found", login)))
				return
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println(err.Error())
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
		}
	}

	if err := goboardcookie.DeleteCookiesForUser(a.db, login); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	return

}

func (a *AdminHandler) deletePost(w http.ResponseWriter, rq *http.Request) {

	postId := (mux.Vars(rq))["id"]

	id, err := strconv.ParseUint(postId, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	if err := goboardbackend.DeletePost(a.db, id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	return

}

func (a *AdminHandler) checkAdminToken(token string) bool {
	return len(a.adminToken) > tokenMinLen && token == a.adminToken
}
