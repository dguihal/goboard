package user

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dguihal/goboard/handlers"
	goboardcookie "github.com/dguihal/goboard/internal/cookie"
	goboarduser "github.com/dguihal/goboard/internal/user"
)

// UserHandler represents the handler of user URLs
type UserHandler struct {
	handlers.GoBoardHandler

	cookieDurationD int
	logger          *log.Logger
}

// NewUserHandler creates an UserHandler object
func NewUserHandler(cookieDuration int) (u *UserHandler) {
	u = &UserHandler{}

	u.logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	u.SupportedOps = []handlers.SupportedOp{
		{PathBase: "/user/add", RestPath: "/user/add", Method: "POST", Handler: u.addUser},         // Add a user
		{PathBase: "/user/login", RestPath: "/user/login", Method: "POST", Handler: u.authUser},    // Authenticate a user
		{PathBase: "/user/logout", RestPath: "/user/logout", Method: "GET", Handler: u.unAuthUser}, // Unauthenticate a user
		{PathBase: "/user/whoami", RestPath: "/user/whoami", Method: "GET", Handler: u.whoAmI},     // Get self account infos
	}

	u.cookieDurationD = cookieDuration

	return
}

func (u *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	for _, op := range u.SupportedOps {
		if r.Method == op.Method && strings.HasPrefix(r.URL.Path, op.PathBase) {
			// Call specific handling method
			op.Handler(w, r)
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

	if passwd = r.FormValue("password"); len(passwd) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Password can't be empty"))
		return
	}

	err := goboarduser.AddUser(u.Db, login, passwd)

	if err == nil {
		// User created : Send him a cookie
		if cookie, err := goboardcookie.ForUser(u.Db, login, u.cookieDurationD); err == nil {
			http.SetCookie(w, &cookie)
			w.WriteHeader(http.StatusCreated)
			return
		}
	} else {
		if uerr, ok := err.(*goboarduser.Error); ok {
			if uerr.ErrCode == goboarduser.UserAlreadyExistsError {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte("User login already exists"))
				return
			}
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
	u.logger.Println(err.Error())
}

func (u *UserHandler) authUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var login, passwd string = "", ""
	if login = r.FormValue("login"); len(login) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Login can't be empty"))
		return
	}

	if passwd = r.FormValue("password"); len(passwd) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Password can't be empty"))
		return
	}

	if err := goboarduser.AuthUser(u.Db, login, passwd); err != nil {
		u.logger.Println(err.Error())
		if uerr, ok := err.(*goboarduser.Error); ok {
			switch uerr.ErrCode {
			case goboarduser.AuthenticationFailed:
				w.WriteHeader(http.StatusUnauthorized)
			case goboarduser.DatabaseError:
				w.WriteHeader(http.StatusInternalServerError)
				u.logger.Println(err.Error())
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			u.logger.Println(err.Error())
		}
	} else {
		var cookie http.Cookie
		var user goboarduser.User
		var userJSON []byte
		var err error

		// User authenticated : Send him a cookie
		if cookie, err = goboardcookie.ForUser(u.Db, login, u.cookieDurationD); err == nil {
			if user, err = goboarduser.GetUser(u.Db, login); err == nil {
				userJSON, err = json.Marshal(user)
			}
		}

		if err == nil {
			http.SetCookie(w, &cookie)
			w.WriteHeader(http.StatusOK)
			w.Write(userJSON)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			u.logger.Println(err.Error())
		}
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
	var login = ""

	for _, c := range r.Cookies() {
		login, _ = goboardcookie.LoginForCookie(u.Db, c)
		if len(login) > 0 {
			break
		}
	}

	if len(login) > 0 {
		var err error
		var user goboarduser.User

		if user, err = goboarduser.GetUser(u.Db, login); err == nil {
			var data []byte

			if data, err = json.Marshal(user); err == nil {
				w.WriteHeader(http.StatusOK)
				w.Write(data)
				return
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
		u.logger.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("You need to be authenticated"))
}
