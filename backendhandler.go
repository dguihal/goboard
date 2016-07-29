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
	"tsv":  true,
	"json": true,
}

var knownHeaders = map[string]string{
	"application/xml":  "xml",
	"text/xml":         "xml",
	"application/json": "json",
	"text/tsv":         "tsv",
}

type BackendHandler struct {
	GoboardHandler

	historySize int
}

func NewBackendHandler(db *bolt.DB, historySize int) (b *BackendHandler) {
	b = &BackendHandler{}

	b.db = db

	b.supportedOps = []SupportedOp{
		{"/backend", "/backend", "GET", b.getBackend},          // Get backend (in xml)
		{"/backend", "/backend/{format}", "GET", b.getBackend}, // Get backend (in specific format)
		{"/post", "/post", "POST", b.post},                     // Post new message
		{"/post/", "/post/{id}", "GET", b.getPost},             // Get a specific message (in xml)
		{"/post/", "/post/{id}/{format}", "GET", b.getPost},    // Get a specific message (in specific format)
	}

	b.historySize = historySize
	return
}

func (b *BackendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	for _, op := range b.supportedOps {
		if r.Method == op.Method && strings.HasPrefix(r.URL.Path, op.PathBase) {
			// Call specific handling method
			op.handler(w, r)
			return
		}
	}

	// If we are here : not methods has been found (shouldn't happen)
	w.WriteHeader(http.StatusNotFound)
	return
}

func (b *BackendHandler) getBackend(w http.ResponseWriter, r *http.Request) {

	lastStr := r.URL.Query().Get("last")
	last, err := strconv.ParseUint(lastStr, 10, 64)
	if err != nil {
		last = 0
	}

	posts, err := goboardbackend.GetBackend(b.db, b.historySize, last)

	if err == nil {

		vars := mux.Vars(r)
		format := guessFormat(vars["format"], r.Header.Get("Accept"))

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

func (b *BackendHandler) getPost(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 64)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing required post id as unsigned int PATH variable"))
		return
	}

	post, err := goboardbackend.GetPost(b.db, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if post.Id == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var data []byte
	format := guessFormat(vars["format"], r.Header.Get("Accept"))
	switch format {
	case "json":
		data, err = json.Marshal(post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case "tsv":
		str := fmt.Sprintf("%d\t%s\t%s\t%s\t%s\n",
			post.Id, post.Time.Format(goboardbackend.PostTimeFormat), post.Info, post.Login, post.Message)
		data = []byte(str)
	default:
		data, err = xml.Marshal(post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)

	return
}

func (b *BackendHandler) post(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	bP := bluemonday.NewPolicy()
	bP.AllowStandardURLs()
	bP.AllowAttrs("href").OnElements("a")
	bP.AllowElements("i")
	bP.AllowElements("u")
	bP.AllowElements("b")
	bP.AllowElements("s")
	bP.AllowElements("em")
	bP.AllowElements("tt")

	message := bP.Sanitize(r.FormValue("message"))
	login := ""

	if cookies := r.Cookies(); len(cookies) > 0 {
		var err error

		if login, err = cookie.LoginForCookie(b.db, cookies[0].Value); err != nil {
			fmt.Println("POST :", err.Error())
			login = ""
		}
	}

	// Build Post object to store
	p := goboardbackend.Post{
		Time:    goboardbackend.PostTime{time.Now()},
		Login:   login,
		Info:    r.Header.Get("User-Agent"),
		Message: message,
	}

	// Try to store it
	if postId, err := goboardbackend.PostMessage(b.db, p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	} else {
		w.Header().Set("X-Post-Id", strconv.FormatUint(postId, 10))
		w.WriteHeader(http.StatusOK)
	}

	return
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
