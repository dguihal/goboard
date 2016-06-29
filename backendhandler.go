package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	goboardbackend "github.com/dguihal/goboard/backend"
	"github.com/dguihal/goboard/cookie"
	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
)

type BackendHandler struct {
	GoboardHandler

	historySize int
}

func NewBackendHandler(db *bolt.DB, historySize int) (r *BackendHandler) {
	r = &BackendHandler{}

	r.db = db

	r.supportedOps = []SupportedOp{
		{"/backend", "GET"},          // Get backend (in xml)
		{"/backend/{format}", "GET"}, //Get backend (in specific format)
		{"/post", "POST"},            // Post new message
	}

	r.historySize = historySize
	return
}

func (r *BackendHandler) Post(post goboardbackend.Post) (postId uint64, err error) {
	postId, err = goboardbackend.PostMessage(r.db, post)
	return
}

func (r *BackendHandler) Get(last uint64) (posts []goboardbackend.Post, err error) {
	posts, err = goboardbackend.GetBackend(r.db, r.historySize, last)
	return
}

func (r *BackendHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	switch rq.Method {
	case "POST":
		fmt.Println("POST")
		rq.ParseForm()

		bP := bluemonday.NewPolicy()
		bP.AllowStandardURLs()
		bP.AllowAttrs("href").OnElements("a")
		bP.AllowElements("i")
		bP.AllowElements("u")
		bP.AllowElements("b")
		bP.AllowElements("s")
		bP.AllowElements("em")
		bP.AllowElements("tt")

		message := bP.Sanitize(rq.FormValue("message"))
		login := ""

		// TODO : Get the session cookie and fetch the corresponding user
		cookies := rq.Cookies()

		if len(cookies) > 0 {
			var err error

			if login, err = cookie.LoginForCookie(r.db, cookies[0].Value); err != nil {
				fmt.Println("POST :", err.Error())
				login = ""
			}
		}

		p := goboardbackend.Post{
			Time:    goboardbackend.PostTime{time.Now()},
			Login:   login,
			Info:    rq.Header.Get("User-Agent"),
			Message: message,
		}

		postId, err := r.Post(p)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.Header().Set("X-Post-Id", strconv.FormatUint(postId, 10))
			w.WriteHeader(http.StatusOK)
		}
	case "GET":
		fmt.Println("GET")

		lastAttrStr := rq.URL.Query().Get("last")
		lastAttr, err := strconv.ParseUint(lastAttrStr, 10, 64)
		if err != nil {
			lastAttr = 0
		}

		posts, err := r.Get(lastAttr)
		if err == nil {
			vars := mux.Vars(rq)
			format := vars["format"]

			var data []byte

			if format == "" || format == "xml" {
				data = postsToXml(posts)
				w.Header().Set("Content-Type", "application/xml")
			} else if format == "json" {
				data = postsToJson(posts)
				w.Header().Set("Content-Type", "application/json")
			} else if format == "tsv" {
				data = postsToTsv(posts)
				w.Header().Set("Content-Type", "text/tab-separated-values")
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(data))
			w.Write([]byte("\n"))

		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

func postsToXml(posts []goboardbackend.Post) []byte {
	var b = goboardbackend.Board{}
	b.Site = "http://localhost"

	var index int = 0
	for _, post := range posts {
		if post.Id == 0 {
			break
		}
		index++
	}

	var pmin = make([]goboardbackend.Post, index)
	copy(pmin, posts)
	b.Posts = pmin

	s, err := xml.Marshal(b)
	if err != nil {
		return []byte(err.Error())
	}
	return s
}

func postsToJson(posts []goboardbackend.Post) []byte {
	return []byte("")
}

func postsToTsv(posts []goboardbackend.Post) []byte {
	var b bytes.Buffer

	for _, post := range posts {
		if post.Id == 0 {
			break
		}
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\t%s\n",
			post.Id, post.Time.Format(goboardbackend.PostTimeFormat), post.Info, post.Login, post.Message)
	}
	return b.Bytes()
}
