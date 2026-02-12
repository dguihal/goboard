package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	goboardcookie "github.com/dguihal/goboard/internal/cookie"
	goboarduser "github.com/dguihal/goboard/internal/user"
)

// UserHandler represents the handler of user URLs
type UserHandler struct {
	GoBoardHandler

	cookieDurationD int
	logger          *log.Logger
}

// NewUserHandler creates an UserHandler object
func NewUserHandler(cookieDuration int) (u *UserHandler) {
	u = &UserHandler{}

	u.logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	u.supportedOps = []SupportedOp{
		{"/user/add", "/user/add", "POST", u.addUser},         // Add a user
		{"/user/login", "/user/login", "POST", u.authUser},    // Authenticate a user
		{"/user/logout", "/user/logout", "GET", u.unAuthUser}, // Unauthenticate a user
		{"/user/whoami", "/user/whoami", "GET", u.whoAmI},     // Get self account infos
	}

	u.cookieDurationD = cookieDuration

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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	login := r.FormValue("login")
	if len(login) == 0 {
		http.Error(w, "Login can't be empty", http.StatusBadRequest)
		return
	}

	passwd := r.FormValue("password")
	if len(passwd) == 0 {
		http.Error(w, "Password can't be empty", http.StatusBadRequest)
		return
	}

	if err := goboarduser.AddUser(u.Db, login, passwd); err != nil {
		if uerr, ok := err.(*goboarduser.Error); ok {
			if uerr.ErrCode == goboarduser.UserAlreadyExistsError {
				http.Error(w, "User login already exists", http.StatusConflict)
				return
			}
		}
		u.logger.Println(err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// User created: Send him a cookie
	if cookie, err := goboardcookie.ForUser(u.Db, login, u.cookieDurationD); err == nil {
		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	} else {
		u.logger.Printf("User created, but failed to create cookie for %s: %v", login, err)
		http.Error(w, "User created, but failed to generate session", http.StatusInternalServerError)
	}
}

func (u *UserHandler) authUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	login := r.FormValue("login")
	if len(login) == 0 {
		http.Error(w, "Login can't be empty", http.StatusBadRequest)
		return
	}

	passwd := r.FormValue("password")
	if len(passwd) == 0 {
		http.Error(w, "Password can't be empty", http.StatusBadRequest)
		return
	}

	if err := goboarduser.AuthUser(u.Db, login, passwd); err != nil {
		u.logger.Println(err.Error())
		if uerr, ok := err.(*goboarduser.Error); ok && uerr.ErrCode == goboarduser.AuthenticationFailed {
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
		} else {
			http.Error(w, "Internal server error during authentication", http.StatusInternalServerError)
		}
		return
	}

	// User authenticated: Get user data, cookie, and marshal to JSON
	user, err := goboarduser.GetUser(u.Db, login)
	if err != nil {
		u.logger.Printf("Auth successful, but failed to get user %s: %v", login, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		u.logger.Printf("Failed to marshal user data for %s: %v", login, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cookie, err := goboardcookie.ForUser(u.Db, login, u.cookieDurationD)
	if err != nil {
		u.logger.Printf("Failed to create cookie for user %s: %v", login, err)
		http.Error(w, "Failed to generate session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &cookie)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(userJSON); err != nil {
		u.logger.Printf("Failed to write user JSON response: %v", err)
	}
}

func (u *UserHandler) unAuthUser(w http.ResponseWriter, r *http.Request) {

	for _, cookie := range r.Cookies() {
		cookie.MaxAge = 0
		cookie.Value = ""
		cookie.Path = "/"
		http.SetCookie(w, cookie)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (u *UserHandler) whoAmI(w http.ResponseWriter, r *http.Request) {
	var login string
	for _, c := range r.Cookies() {
		// Check for a valid login from any of the cookies
		l, err := goboardcookie.LoginForCookie(u.Db, c)
		if err == nil && len(l) > 0 {
			login = l
			break
		}
	}

	if len(login) == 0 {
		http.Error(w, "You need to be authenticated", http.StatusForbidden)
		return
	}

	user, err := goboarduser.GetUser(u.Db, login)
	if err != nil {
		u.logger.Printf("Could not get user data for authenticated user %s: %v", login, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(user)
	if err != nil {
		u.logger.Printf("Could not marshal user data for %s: %v", login, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		u.logger.Printf("Failed to write whoAmI response: %v", err)
	}
}
