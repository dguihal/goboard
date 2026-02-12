// Package cookie provides management of cookies in database
package cookie

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dchest/uniuri"
	bolt "go.etcd.io/bbolt"
)

const goboardCookieName string = "goboard_id"
const usersCookieBucketName string = "UsersCookie"

// UserCookie struct used to modelize a cookie
type UserCookie struct {
	Login  string
	Cookie http.Cookie
}

// A list of error codes used in UserCookieError
const (
	NoError       = iota
	DatabaseError = iota
	NoCookieFound = iota
)

// UserCookieError a struct modelizing cookie operation errors
type UserCookieError struct {
	error
	ErrCode int // Error Code
}

func (e *UserCookieError) Error() string { return e.error.Error() }

// ForUser returns a valid cookie (already existing or new) for a user
func ForUser(db *bolt.DB, login string, cookieDurationD int) (cookie http.Cookie, err error) {

	if cookie, err = fetchCookieForUser(db, login); err != nil {
		if ucerr, ok := err.(*UserCookieError); ok {
			if ucerr.ErrCode == NoCookieFound {
				// No existing valid cookie found, create one
				cookie, err = createAndStoreCookie(db, login, cookieDurationD)
			}
		}
	}

	return
}

// createAndStoreCookie creates a new cookie and stores it in database
func createAndStoreCookie(db *bolt.DB, login string, cookieDurationD int) (cookie http.Cookie, err error) {

	expiration := time.Now().Add(time.Duration(cookieDurationD) * 24 * time.Hour)
	cookie = http.Cookie{
		Name:     goboardCookieName,
		Value:    uniuri.NewLen(64),
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
	}

	uc := UserCookie{login, cookie}

	buf, err := json.Marshal(uc)
	if err != nil {
		ucerr := &UserCookieError{error: err, ErrCode: DatabaseError}
		cookie = http.Cookie{}
		fmt.Println(err.Error())
		return cookie, ucerr
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersCookieBucketName))
		if err != nil {
			ucerr := &UserCookieError{error: err, ErrCode: DatabaseError}
			cookie = http.Cookie{}
			fmt.Println(err.Error())
			return ucerr
		}

		err = b.Put([]byte(cookie.Value), buf)
		if err != nil {
			ucerr := &UserCookieError{error: err, ErrCode: DatabaseError}
			cookie = http.Cookie{}
			fmt.Println(err.Error())
			return ucerr
		}

		return nil
	})

	return
}

// DeleteCookiesForUser delete stored cookies for user
func DeleteCookiesForUser(db *bolt.DB, login string) (err error) {

	err = db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(usersCookieBucketName))
		if err != nil {
			uerr := &UserCookieError{error: err, ErrCode: DatabaseError}
			return uerr
		}

		c := b.Cursor()
		userCookie := UserCookie{}

		for k, v := c.First(); k != nil; k, v = c.Next() {
			if uErr := json.Unmarshal(v, &userCookie); uErr != nil {
				// Log and skip corrupted data
				log.Printf("could not unmarshal cookie, skipping: %v", uErr)
				continue
			}

			if userCookie.Login == login {
				if err = b.Delete(k); err != nil {
					uerr := &UserCookieError{error: err, ErrCode: DatabaseError}
					return uerr
				}
			}
		}

		return nil
	})

	return
}

// fetchCookieForUser retreive a valid stored cookie for a user (if any in database)
func fetchCookieForUser(db *bolt.DB, login string) (cookie http.Cookie, err error) {

	cookie = http.Cookie{}

	err = db.Update(func(tx *bolt.Tx) error {
		// Find if non expired cookie already exists
		b, err := tx.CreateBucketIfNotExists([]byte(usersCookieBucketName))
		if err != nil {
			ucerr := &UserCookieError{error: err, ErrCode: DatabaseError}
			return ucerr
		}

		c := b.Cursor()
		userCookie := UserCookie{}
		cookieFound := false
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if err := json.Unmarshal(v, &userCookie); err != nil {
				log.Printf("Could not unmarshal cookie data, skipping: %v", err)
				continue
			}

			if userCookie.Login == login {
				if userCookie.Cookie.Expires.Before(time.Now()) {
					// Delete expired cookie
					if err := b.Delete(k); err != nil {
						return &UserCookieError{error: err, ErrCode: DatabaseError}
					}
				} else {
					if !cookieFound {
						// Pick up valid cookie
						cookie = userCookie.Cookie
						cookieFound = true
					} else {
						// Remove duplicates
						if err := b.Delete(k); err != nil {
							return &UserCookieError{error: err, ErrCode: DatabaseError}
						}
					}
				}
			}
		}

		if !cookieFound {
			ucerr := &UserCookieError{error: fmt.Errorf("no cookie found"), ErrCode: NoCookieFound}
			return ucerr
		}

		return nil
	})

	return
}

// LoginForCookie get the user associated with a cookie
func LoginForCookie(db *bolt.DB, cookie *http.Cookie) (login string, err error) {
	var uc = UserCookie{}
	login = ""

	if cookie.Name != goboardCookieName || cookie.Expires.Before(time.Now()) {
		return
	}

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(usersCookieBucketName))

		if b == nil {
			return nil
		}

		v := b.Get([]byte(cookie.Value))
		if v != nil {
			return json.Unmarshal(v, &uc)
		}
		return nil
	})

	if len(uc.Login) > 0 {
		login = uc.Login

		if err == nil && uc.Cookie.Expires.Before(time.Now()) {
			err = db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(usersCookieBucketName))
				if b == nil {
					return nil
				}

				return b.Delete([]byte(cookie.Value))
			})
		}
	}
	return
}
