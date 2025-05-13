package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dguihal/goboard/handlers"
	goboardbackend "github.com/dguihal/goboard/internal/backend"
	goboardcookie "github.com/dguihal/goboard/internal/cookie"
	goboarduser "github.com/dguihal/goboard/internal/user"
	"github.com/gorilla/mux"
)

const tokenMinLen int = 0
const tokenWarnLen int = 12

// AdminHandler represents the handler of admin URLs
type AdminHandler struct {
	handlers.GoBoardHandler

	adminToken string
}

// NewAdminHandler creates an AdminHandler object
func NewAdminHandler(adminToken string) (a *AdminHandler) {
	a = &AdminHandler{}

	a.SupportedOps = []handlers.SupportedOp{
		{PathBase: "/admin/user/", RestPath: "/admin/user/{login}", Method: "DELETE", Handler: a.deleteUser}, // Delete a user
		{PathBase: "/admin/user/", RestPath: "/admin/user/{login}", Method: "GET", Handler: a.getUser},       // Get a user info
		{PathBase: "/admin/post/", RestPath: "/admin/post/{id}", Method: "DELETE", Handler: a.deletePost},    // Delete a post
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

	for _, op := range a.SupportedOps {
		if r.Method == op.Method && strings.HasPrefix(r.URL.Path, op.PathBase) {
			// Call specific handling method
			op.Handler(w, r)
			return
		}
	}

	// If we are here : no methods has been found (shouldn't happen)
	w.WriteHeader(http.StatusNotFound)
}

func (a *AdminHandler) deleteUser(w http.ResponseWriter, r *http.Request) {

	login := (mux.Vars(r))["login"]

	if err := goboarduser.DeleteUser(a.Db, login); err != nil {
		if uerr, ok := err.(*goboarduser.Error); ok {
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
