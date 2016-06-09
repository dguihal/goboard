package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/boltdb/bolt"
	"golang.org/x/crypto/bcrypt"
)

const usersBucketName string = "Users"

type User struct {
	Id             uint64
	Login          string
	HashedPassword []byte
}

const (
	noError           = iota
	internalDbError   = iota
	userAlreadyExists = iota
)

type userDBError struct {
	Code uint
	Err  error
}

type userHandler struct {
	db *bolt.DB
}

func newUserHandler(db *bolt.DB) (u *userHandler) {
	u = &userHandler{}
	u.db = db

	return
}

func (u *userHandler) AddUser(login string, password string) (err userDBError) {
	var errCode uint = 0
	error := u.db.Batch(func(tx *bolt.Tx) error {
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
			return err
		}

		user = User{Id: uint64(id), Login: login, HashedPassword: hashedPassword}

		buf, err := json.Marshal(user)
		if err != nil {
			return err
		}

		err = b.Put(itob(user.Id), buf)

		return nil
	})

	ue := userDBError{Code: noError, Err: nil}
	if error != nil {
		ue.Code = errCode
		ue.Err = error
	}
	return ue
}

func (u *userHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		fmt.Println("POST")
		r.ParseForm()

		loginAttr := r.FormValue("login")
		passwdAttr := r.FormValue("password")

		fmt.Println("POST", loginAttr, passwdAttr)

		dbErr := u.AddUser(loginAttr, passwdAttr)
		if dbErr.Code != noError {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(dbErr.Err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}
