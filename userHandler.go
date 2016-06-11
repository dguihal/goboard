package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dchest/uniuri"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id             uint64
	Login          string
	HashedPassword []byte
	salt           []byte
}

type UserCookie struct {
	Login  string
	Cookie http.Cookie
}

const (
	noError           = iota
	internalDbError   = iota
	internalError     = iota
	userAlreadyExists = iota
	userDoesNotExists = iota
	wrongPassword     = iota
)

type userHandler struct {
	db *bolt.DB

	cookieDuration_d int
}

func newUserHandler(db *bolt.DB, cookieDuration int) (u *userHandler) {
	u = &userHandler{}
	u.db = db
	u.cookieDuration_d = cookieDuration

	return
}

func (u *userHandler) AddUser(w http.ResponseWriter, login string, password string) {

	var errCode int = noError

	err := u.db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			errCode = internalDbError
			return err
		}

		c := b.Cursor()
		user := User{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			json.Unmarshal(v, &user)

			if user.Login == login {
				errCode = userAlreadyExists
				err := errors.New("User already exists")
				return err
			}
		}

		// Else we have a really new user
		id, _ := b.NextSequence()
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		fmt.Println("AddUser", password, hashedPassword)
		if err != nil {
			errCode = internalError
			return err
		}

		user = User{Id: uint64(id), Login: login, HashedPassword: hashedPassword}

		buf, err := json.Marshal(user)
		if err != nil {
			errCode = internalError
			return err
		}

		err = b.Put(itob(user.Id), buf)

		errCode = noError
		return nil
	})

	if errCode != noError {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return
}

func (u *userHandler) AuthUser(w http.ResponseWriter, login string, password string) {

	var errCode int = noError

	err := u.db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersCookieBucketName))

		c := b.Cursor()
		user := User{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			json.Unmarshal(v, &user)

			if user.Login == login {
				err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
				if err != nil {
					errCode = wrongPassword
				}
				return err
			}
		}

		expiration := time.Now().Add(time.Duration(u.cookieDuration_d) * 24 * time.Hour)
		cookie := http.Cookie{Name: goboard_cookie_name, Value: uniuri.NewLen(64), Expires: expiration}

		userCookie := UserCookie{user.Login, cookie}
		buf, err := json.Marshal(userCookie)
		if err != nil {
			errCode = internalError
			return err
		}

		err = b.Put([]byte(user.Login), buf)

		return nil
	})

	if errCode != noError {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	} else {
		expiration := time.Now().Add(time.Duration(u.cookieDuration_d) * 24 * time.Hour)
		cookie := http.Cookie{Name: goboard_cookie_name, Value: uniuri.NewLen(64), Expires: expiration}
		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	}

	return
}

func (u *userHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		fmt.Println("POST")
		r.ParseForm()

		loginAttr := r.FormValue("login")
		passwdAttr := r.FormValue("password")

		fmt.Println("POST", loginAttr, passwdAttr)

		if strings.HasSuffix(r.URL.Path, "add") {
			u.AddUser(w, loginAttr, passwdAttr)
		} else if strings.HasSuffix(r.URL.Path, "login") {
			u.AuthUser(w, loginAttr, passwdAttr)
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}
}
