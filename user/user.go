// user.go
package user

import (
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
	"golang.org/x/crypto/bcrypt"
)

const usersBucketName string = "Users"

const (
	NoError                = iota
	UserAlreadyExistsError = iota
	UserDoesNotExistsError = iota
	DatabaseError          = iota
	AuthenticationFailed   = iota
)

type UserError struct {
	error
	ErrCode int // Error Code
}

func (e *UserError) Error() string { return e.error.Error() }

type User struct {
	Login          string
	HashedPassword []byte
	salt           []byte
}

func AddUser(db *bolt.DB, login string, password string) (uerr error) {

	db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return uerr
		}

		v := b.Get([]byte(login))
		if v != nil {
			uerr = &UserError{error: fmt.Errorf("User already exists"), ErrCode: UserAlreadyExistsError}
			return uerr
		}

		// Else we have a really new user
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return uerr
		}

		user := User{Login: login, HashedPassword: hashedPassword}

		buf, err := json.Marshal(user)
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return uerr
		}

		err = b.Put([]byte(user.Login), buf)
		if err != nil {
			uerr = &UserError{error: err, ErrCode: DatabaseError}
			return uerr
		}

		return nil
	})

	return
}

func AuthUser(db *bolt.DB, login string, password string) (uerr error) {

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(usersBucketName))

		v := b.Get([]byte(login))

		if v == nil { // User does not exists
			uerr = &UserError{error: fmt.Errorf("Authentification failed"), ErrCode: AuthenticationFailed}
			return uerr
		} else {
			user := User{}
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

func DeleteUser(db *bolt.DB, login string) (err error) {

	err = db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			uerr := &UserError{error: err, ErrCode: DatabaseError}
			return uerr
		}

		v := b.Get([]byte(login))

		if v == nil { // User does not exists
			uerr := &UserError{error: fmt.Errorf("User does not exists"), ErrCode: UserDoesNotExistsError}
			return uerr
		} else {
			if err = b.Delete([]byte(login)); err != nil {
				uerr := &UserError{error: err, ErrCode: DatabaseError}
				return uerr
			}
		}

		return nil
	})

	return
}
