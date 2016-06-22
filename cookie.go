// cookie.go
package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dchest/uniuri"
)

const goboard_cookie_name string = "goboard_id"
const usersCookieBucketName string = "UsersCookie"

type UserCookie struct {
	Login  string
	Cookie http.Cookie
}

type UserCookieError struct {
	error
	ErrCode int // Error Code
}

func (e *UserCookieError) Error() string { return e.error.Error() }

/*

	b, err = tx.CreateBucketIfNotExists([]byte(usersCookieBucketName))
	if err != nil {
		userError := UserError{Msg: err.Error(), ErrCode: DatabaseError}
		return err
	}

	cookie, err = createAndStoreCookie(b, login, u.cookieDuration_d)
	if err != nil {
		userError := UserError{Msg: err.Error(), ErrCode: DatabaseError}
		return err
	}
*/

func CreateAndStoreCookie(db *bolt.DB, login string, cookieDuration_d int) (cookie http.Cookie, err error) {

	expiration := time.Now().Add(time.Duration(cookieDuration_d) * 24 * time.Hour)
	cookie = http.Cookie{Name: goboard_cookie_name, Value: uniuri.NewLen(64), Expires: expiration}

	uc := UserCookie{login, cookie}

	buf, err := json.Marshal(uc)
	if err != nil {
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(usersCookieBucketName))
		if b == nil {
			return nil
		}

		err = b.Put([]byte(cookie.Value), buf)

		return nil
	})

	return
}

func LoginForCookie(db *bolt.DB, cookieValue string) (login string, err error) {
	var uc = UserCookie{}
	login = ""

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(usersCookieBucketName))
		if b == nil {
			return nil
		}

		v := b.Get([]byte(cookieValue))
		if v != nil {
			json.Unmarshal(v, &uc)
		}
		return nil
	})

	login = uc.Login

	if err == nil && login != "" && uc.Cookie.Expires.Before(time.Now()) {
		err = db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(usersCookieBucketName))
			if b == nil {
				return nil
			}

			b.Delete([]byte(cookieValue))

			return nil
		})
	}
	return
}
