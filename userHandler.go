package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"
	"golang.org/x/crypto/bcrypt"
)

const usersBucketName string = "Users"

type User struct {
	Id             uint64
	Login          string
	HashedPassword []byte
	salt           []byte
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
}

func newUserHandler(db *bolt.DB) (u *userHandler) {
	u = &userHandler{}
	u.db = db

	return
}

func (u *userHandler) AddUser(login string, password string) (errCode uint, err error) {

	err = u.db.Batch(func(tx *bolt.Tx) error {
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

	return
}

func (u *userHandler) AuthUser(login string, password string) (errCode uint, err error) {

	err = u.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(usersBucketName))

		c := b.Cursor()
		user := User{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			json.Unmarshal(v, &user)

			if user.Login == login {
				err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
				if err != nil {
					errCode = wrongPassword
				}
				return err
			}
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

		fmt.Println("POST", loginAttr, passwdAttr)

		var errCode uint = noError
		err := errors.New("")

		if strings.HasSuffix(r.URL.Path, "add") {
			errCode, err = u.AddUser(loginAttr, passwdAttr)
		} else if strings.HasSuffix(r.URL.Path, "login") {
			errCode, err = u.AuthUser(loginAttr, passwdAttr)
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if errCode != noError {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}
