// cookie.go
package cookie

import (
	"encoding/json"
	"fmt"
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

const (
	NoError       = iota
	DatabaseError = iota
	NoCookieFound = iota
)

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

func CookieForUser(db *bolt.DB, login string, cookieDuration_d int) (cookie http.Cookie, err error) {

	if cookie, err = FetchCookieForUser(db, login); err != nil {
		if ucerr, ok := err.(*UserCookieError); ok {
			if ucerr.ErrCode == NoCookieFound {
				// No existing valid cookie found, create one
				cookie, err = CreateAndStoreCookie(db, login, cookieDuration_d)
			}
		}
	}

	return
}

func CreateAndStoreCookie(db *bolt.DB, login string, cookieDuration_d int) (cookie http.Cookie, err error) {

	expiration := time.Now().Add(time.Duration(cookieDuration_d) * 24 * time.Hour)
	cookie = http.Cookie{Name: goboard_cookie_name, Value: uniuri.NewLen(64), Expires: expiration}

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

		return nil
	})

	return
}

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
			json.Unmarshal(v, &userCookie)

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

func FetchCookieForUser(db *bolt.DB, login string) (cookie http.Cookie, err error) {

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
			json.Unmarshal(v, &userCookie)

			if userCookie.Login == login {
				if userCookie.Cookie.Expires.Before(time.Now()) {
					// Delete expired cookie
					b.Delete(k)
				} else {
					if !cookieFound {
						// Pick up valid cookie
						cookie = userCookie.Cookie
						cookieFound = true
					} else {
						// Remove duplicates
						b.Delete(k)
					}
					return nil
				}
			}
		}

		if !cookieFound {
			ucerr := &UserCookieError{error: fmt.Errorf("No cookie found"), ErrCode: NoCookieFound}
			return ucerr
		}

		return nil
	})

	return
}

func LoginForCookie(db *bolt.DB, cookieValue string) (login string, err error) {
	var uc = UserCookie{}
	login = ""

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(usersCookieBucketName))

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
				ucerr := &UserCookieError{error: err, ErrCode: DatabaseError}
				return ucerr
			}

			b.Delete([]byte(cookieValue))

			return nil
		})
	}
	return
}
