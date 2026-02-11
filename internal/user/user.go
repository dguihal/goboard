// user.go
package user

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
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

type Error struct {
	error
	ErrCode int // Error Code
}

func (e *Error) Error() string { return e.error.Error() }

type User struct {
	Login          string
	CreationDate   time.Time
	HashedPassword []byte `json:"HashedPassword,omitempty"`
}

func AddUser(db *bolt.DB, login string, password string) (uerr error) {

	uerr = db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			uerr = &Error{error: err, ErrCode: DatabaseError}
			return uerr
		}

		v := b.Get([]byte(login))
		if v != nil {
			uerr = &Error{error: fmt.Errorf("User already exists"), ErrCode: UserAlreadyExistsError}
			return uerr
		}

		// Else we have a really new user
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			uerr = &Error{error: err, ErrCode: DatabaseError}
			return uerr
		}

		user := User{Login: login, HashedPassword: hashedPassword, CreationDate: time.Now()}

		buf, err := json.Marshal(user)
		if err != nil {
			uerr = &Error{error: err, ErrCode: DatabaseError}
			return uerr
		}

		err = b.Put([]byte(user.Login), buf)
		if err != nil {
			uerr = &Error{error: err, ErrCode: DatabaseError}
			return uerr
		}

		return nil
	})

	return
}

func AuthUser(db *bolt.DB, login string, password string) (uerr error) {

	uerr = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(usersBucketName))
		var v []byte

		if b != nil {
			v = b.Get([]byte(login))
		}

		if v == nil { // User does not exists
			uerr = &Error{error: fmt.Errorf("authentification failed"), ErrCode: AuthenticationFailed}
			return uerr
		}
		user := User{}
		if err := json.Unmarshal(v, &user); err != nil {
			uerr = &Error{error: err, ErrCode: DatabaseError}
			return uerr
		}

		if user.Login == login {
			err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
			if err != nil {
				uerr = &Error{error: err, ErrCode: AuthenticationFailed}
				return err
			}
		}

		return nil
	})

	return
}

func DeleteUser(db *bolt.DB, login string) (uerr error) {

	uerr = db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersBucketName))
		if err != nil {
			uerr = &Error{error: err, ErrCode: DatabaseError}
			return err
		}

		v := b.Get([]byte(login))

		if v == nil { // User does not exists
			uerr = &Error{error: fmt.Errorf("User does not exists"), ErrCode: UserDoesNotExistsError}
			return uerr
		} else {
			if err = b.Delete([]byte(login)); err != nil {
				uerr = &Error{error: err, ErrCode: DatabaseError}
				return err
			}
		}

		return nil
	})

	return
}

func GetUser(db *bolt.DB, login string) (user User, uerr error) {

	uerr = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(usersBucketName))
		var v []byte

		if b != nil {
			v = b.Get([]byte(login))
		}

		if v == nil {
			uerr = &Error{error: fmt.Errorf("User does not exists"), ErrCode: UserDoesNotExistsError}
			return uerr
		}

		if err := json.Unmarshal(v, &user); err != nil {
			uerr = &Error{error: err, ErrCode: DatabaseError}
			return uerr
		}
		user.HashedPassword = nil
		return nil
	})

	return
}
