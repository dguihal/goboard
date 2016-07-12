package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	goboardbackend "github.com/dguihal/goboard/backend"
	"github.com/dguihal/goboard/cookie"
	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
)

const defaultFormat string = "xml"

var allowedFormats = map[string]bool{
	"xml":  true,
	"csv":  true,
	"json": true,
}

var knownHeaders = map[string]string{
	"application/xml":  "xml",
	"text/xml":         "xml",
	"application/json": "json",
	"text/csv":         "csv",
}

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

func (r *BackendHandler) GetBackend(w http.ResponseWriter, last uint64, format string) {
	posts, err := goboardbackend.GetBackend(r.db, r.historySize, last)

	if err == nil {

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

		vars := mux.Vars(rq)
		formatAttr := vars["format"]

		format := guessFormat(formatAttr, rq.Header.Get("Accept"))

		r.GetBackend(w, lastAttr, format)
	}
}

// Guess backend format to deliver based on :
// - 1/ Explicit format by url parameter
// - 2/ Accept HTTP header : Simplified version
// - 3/ xml by default
func guessFormat(formatAttr string, acceptHeader string) (format string) {
	format = "xml"

	if allowedFormats[formatAttr] {
		format = formatAttr
	} else {
		var indexes = make([]int, len(knownHeaders))
		var keys = make([]string, len(knownHeaders))

		i := 0
		for k, _ := range knownHeaders {
			indexes[i] = strings.Index(acceptHeader, k)
			keys[i] = k
			i++
		}

		// Find lowest non null
		min_index := 0
		min_val := indexes[0]

		for i = 1; i < len(indexes); i++ {
			if indexes[i] >= 0 && (min_val < 0 || indexes[i] < min_val) {
				min_val = indexes[i]
				min_index = i
			}
		}

		if min_val >= 0 { // At least one match found
			format = knownHeaders[keys[min_index]]
		}
	}

	return
}

func postsToXml(posts []goboardbackend.Post) []byte {
	var b = goboardbackend.Board{}
	b.Site = "http://localhost"

	var i int = 0
	var p goboardbackend.Post
	for i, p = range posts {
		if p.Id == 0 {
			break
		}
	}

	b.Posts = posts[:i]

	s, err := xml.Marshal(b)
	if err != nil {
		return []byte(err.Error())
	}
	return s
}

func postsToJson(posts []goboardbackend.Post) []byte {
	var b = goboardbackend.Board{}
	b.Site = "http://localhost"

	var i int = 0
	var p goboardbackend.Post
	for i, p = range posts {
		if p.Id == 0 {
			break
		}
	}

	b.Posts = posts[:i]

	s, err := json.Marshal(b)
	if err != nil {
		return []byte(err.Error())
	}
	return s
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
