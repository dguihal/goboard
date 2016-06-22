// user.go
package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/boltdb/bolt"
	goboardutils "github.com/dguihal/goboard/utils"
	"golang.org/x/crypto/bcrypt"
)

const usersBucketName string = "Users"

const (
	NoError                = iota
	UserAlreadyExistsError = iota
	DatabaseError          = iota
	AuthenticationFailed   = iota
)

type UserError struct {
	error
	ErrCode int // Error Code
}

func (e *UserError) Error() string { return e.error.Error() }

type User struct {
	Id             uint64
	Login          string
	HashedPassword []byte
	salt           []byte
}

func AddUser(db *bolt.DB, login string, password string) (cookie http.Cookie, uerr error) {

	db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return uerr
		}

		c := b.Cursor()
		user := User{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			json.Unmarshal(v, &user)

			if user.Login == login {
				e := errors.New("User already exists")
				uerr = &UserError{error: err, ErrCode: UserAlreadyExistsError}
				return e
			}
		}

		// Else we have a really new user
		id, _ := b.NextSequence()
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return err
		}

		user = User{Id: uint64(id), Login: login, HashedPassword: hashedPassword}

		buf, err := json.Marshal(user)
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return err
		}

		err = b.Put(goboardutils.IToB(user.Id), buf)
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return err
		}

		return nil
	})

	return
}

func AuthUser(db *bolt.DB, login string, password string) (uerr error) {

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(usersBucketName))

		c := b.Cursor()
		user := User{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			json.Unmarshal(v, &user)

			if user.Login == login {
				err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
				if err != nil {
					uerr = &UserError{error: err, ErrCode: AuthenticationFailed}
					return err
				}
			}
		}
		return nil
	})

	return
}

func DeleteUser(db *bolt.DB, login string, password string) (err error) {
	return
}
