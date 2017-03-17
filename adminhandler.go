package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	goboardbackend "github.com/dguihal/goboard/backend"
	goboardcookie "github.com/dguihal/goboard/cookie"
	goboarduser "github.com/dguihal/goboard/user"
	"github.com/gorilla/mux"
)

const tokenMinLen int = 0
const tokenWarnLen int = 12

// AdminHandler represents the handler of admin URLs
type AdminHandler struct {
	GoBoardHandler

	adminToken string
}

// NewAdminHandler creates an AdminHandler object
func NewAdminHandler(adminToken string) (a *AdminHandler) {
	a = &AdminHandler{}

	a.supportedOps = []SupportedOp{
		{"/admin/user/", "/admin/user/{login}", "DELETE", a.deleteUser}, // Delete a user
		{"/admin/user/", "/admin/user/{login}", "GET", a.getUser},       // Get a user info
		{"/admin/post/", "/admin/post/{id}", "DELETE", a.deletePost},    // Delete a post
	}

	if len(adminToken) <= tokenMinLen {
		log.Println("Admin token empty : for security reasongs, this means that no admin operations will be authorized")
	} else if len(adminToken) < tokenWarnLen {
		log.Println("Admin token len <", tokenWarnLen, ": Come on I'm sure you can do a lot better")
	}
	a.adminToken = adminToken
	a.BasePath = ""
	return
}

func (a *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	reqAdminToken := r.Header.Get("Token-Id")
	if !a.checkAdminToken(reqAdminToken) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	for _, op := range a.supportedOps {
		if r.Method == op.Method && strings.HasPrefix(r.URL.Path, a.BasePath+op.PathBase) {
			// Call specific handling method
			op.handler(w, r)
			return
		}
	}

	// If we are here : no methods has been found (shouldn't happen)
	w.WriteHeader(http.StatusNotFound)
	return
}

func (a *AdminHandler) deleteUser(w http.ResponseWriter, r *http.Request) {

	login := (mux.Vars(r))["login"]

	if err := goboarduser.DeleteUser(a.Db, login); err != nil {
		if uerr, ok := err.(*goboarduser.UserError); ok {
			if uerr.ErrCode == goboarduser.UserDoesNotExistsError {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(fmt.Sprintf("User %s Not found", login)))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
		}
	}

	if err := goboardcookie.DeleteCookiesForUser(a.Db, login); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	return

}

func (a *AdminHandler) deletePost(w http.ResponseWriter, rq *http.Request) {

	postID := (mux.Vars(rq))["id"]

	id, err := strconv.ParseUint(postID, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	if err := goboardbackend.DeletePost(a.Db, id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	return

}

func (a *AdminHandler) getUser(w http.ResponseWriter, r *http.Request) {

	login := (mux.Vars(r))["login"]

	if user, err := goboarduser.GetUser(a.Db, login); err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		data, err := json.Marshal(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}
	}
}

func (a *AdminHandler) checkAdminToken(token string) bool {
	return len(a.adminToken) > tokenMinLen && token == a.adminToken
}
