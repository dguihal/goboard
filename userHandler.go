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

type userHandler struct {
	GoboardHandler

	cookieDuration_d int
}

func newUserHandler(db *bolt.DB, cookieDuration int) (u *userHandler) {
	u = &userHandler{}
	u.db = db

	u.supportedOps = []SupportedOp{
		{"/user/add", "POST"},   // Add a user
		{"/user/login", "POST"}, // Sign in a user
	}

	u.cookieDuration_d = cookieDuration

	return
}

func (u *userHandler) AddUser(w http.ResponseWriter, login string, password string) {

	var cookie http.Cookie

	err := u.db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		c := b.Cursor()
		user := User{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			json.Unmarshal(v, &user)

			if user.Login == login {
				w.WriteHeader(http.StatusConflict)
				err := errors.New("User already exists")
				w.Write([]byte(err.Error()))
				return err
			}
		}

		// Else we have a really new user
		id, _ := b.NextSequence()
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		user = User{Id: uint64(id), Login: login, HashedPassword: hashedPassword}

		buf, err := json.Marshal(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		err = b.Put(itob(user.Id), buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		b, err = tx.CreateBucketIfNotExists([]byte(usersCookieBucketName))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		cookie, err = createAndStoreCookie(b, login, u.cookieDuration_d)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		return nil
	})

	if err == nil {
		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	}

	return
}

func (u *userHandler) AuthUser(w http.ResponseWriter, login string, password string) {

	var cookie http.Cookie

	err := u.db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		c := b.Cursor()
		user := User{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			json.Unmarshal(v, &user)

			if user.Login == login {
				err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
				if err != nil {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(err.Error()))
					return err
				}

				// Find if non expired cookie already exists
				b, err := tx.CreateBucketIfNotExists([]byte(usersCookieBucketName))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return err
				}
				c := b.Cursor()
				userCookie := UserCookie{}
				for k, v := c.First(); k != nil; k, v = c.Next() {
					json.Unmarshal(v, &userCookie)

					if userCookie.Login == login {
						if userCookie.Cookie.Expires.Before(time.Now()) {
							b.Delete(k)
						} else {
							cookie = userCookie.Cookie
							return nil
						}
					}
				}

				// No Valid cookie found, create one
				cookie, err = createAndStoreCookie(b, login, u.cookieDuration_d)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return err
				}
			}
		}
		return nil
	})

	if err == nil {
		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	}

	return
}

func createAndStoreCookie(b *bolt.Bucket, login string, cookieDuration_d int) (cookie http.Cookie, err error) {

	expiration := time.Now().Add(time.Duration(cookieDuration_d) * 24 * time.Hour)
	cookie = http.Cookie{Name: goboard_cookie_name, Value: uniuri.NewLen(64), Expires: expiration}

	uc := UserCookie{login, cookie}

	buf, err := json.Marshal(uc)
	if err != nil {
		return
	}

	err = b.Put([]byte(cookie.Value), buf)
	return
}

func (u *userHandler) GetUser(cookie string) (login string) {

	u.db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(usersCookieBucketName))
		if b == nil {
			return nil
		}

		return nil
	})

	return
}

func (u *userHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
