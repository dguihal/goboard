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

	goboardbackend "github.com/dguihal/goboard/internal/backend"
	goboardcookie "github.com/dguihal/goboard/internal/cookie"
	"github.com/gorilla/mux"
)

var allowedFormats = map[string]bool{
	"xml":  true,
	"tsv":  true,
	"json": true,
	"raw":  true,
}

var knownHeaders = map[string]string{
	"application/xml":  "xml",
	"text/xml":         "xml",
	"application/json": "json",
	"text/tsv":         "tsv",
}

// BackendHandler represents the handler of backend URLs
type BackendHandler struct {
	GoBoardHandler

	historySize int
}

// NewBackendHandler creates an BackendHandler object
func NewBackendHandler(historySize int, frontLocation string) (b *BackendHandler) {
	b = &BackendHandler{}

	b.supportedOps = []SupportedOp{
		{"/backend", "/backend", "GET", b.getBackend},          // Get backend (in xml)
		{"/backend", "/backend/{format}", "GET", b.getBackend}, // Get backend (in specific format)
		{"/post", "/post", "POST", b.post},                     // Post new message
		{"/post/", "/post/{id}", "GET", b.getPost},             // Get a specific message (in xml)
		{"/post/", "/post/{id}/{format}", "GET", b.getPost},    // Get a specific message (in specific format)
	}

	if location, err := time.LoadLocation(frontLocation); err == nil {
		goboardbackend.TZLocation = location
	} else {
		//Falls back to current Location
		goboardbackend.TZLocation = time.Now().Location()
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
}

func (b *BackendHandler) getBackend(w http.ResponseWriter, r *http.Request) {

	lastStr := r.URL.Query().Get("last")
	last, err := strconv.ParseUint(lastStr, 10, 64)
	if err != nil {
		last = 0
	}

	posts, err := goboardbackend.GetBackend(b.Db, b.historySize, last)

	if err == nil {
		if len(posts) == 0 || posts[0].ID == 0 {
			w.WriteHeader(http.StatusNoContent)
		} else {
			vars := mux.Vars(r)
			format := guessFormat(vars["format"], r.Header.Get("Accept"))

			var data []byte

			if format == "" || format == "xml" {
				data = postsToXML(posts, r.Header.Get("Location"))
				w.Header().Set("Content-Type", "application/xml")
			} else if format == "json" {
				data = postsToJSON(posts)
				w.Header().Set("Content-Type", "application/json")
			} else if format == "tsv" {
				data = postsToTsv(posts)
				w.Header().Set("Content-Type", "text/tab-separated-values")
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(data))
			w.Write([]byte("\n"))
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

// TODO : Manage returning an original posted data for a specific id as text
//        Maybe consider allowing this only for admins (not sure it is relevant)
func (b *BackendHandler) getPost(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseUint(idStr, 10, 64)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing required post id as unsigned int PATH variable"))
		return
	}

	post, err := goboardbackend.GetPost(b.Db, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if post.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var data []byte
	format := guessFormat(vars["format"], r.Header.Get("Accept"))
	switch format {
	case "json":
		post.RawMessage = ""
		data, err = json.Marshal(post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
	case "tsv":
		str := fmt.Sprintf("%d\t%s\t%s\t%s\t%s\n",
			post.ID, post.Time.Format(goboardbackend.PostTimeFormat), post.Info, post.Login, post.Message)
		data = []byte(str)
		w.Header().Set("Content-Type", "text/tab-separated-values")
	case "raw":
		data = []byte(post.RawMessage)
		w.Header().Set("Content-Type", "text/plain")

	default:
		post.RawMessage = ""
		data, err = xml.Marshal(post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (b *BackendHandler) post(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	message, err := goboardbackend.SanitizeAndValidate(r.FormValue("message"))
	// Validation failed
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	rawInfo := r.FormValue("info")
	if len(rawInfo) == 0 {
		rawInfo = r.Header.Get("User-Agent")
	}
	info := goboardbackend.Sanitize(rawInfo)
	login := ""

	for _, c := range r.Cookies() {
		login, _ = goboardcookie.LoginForCookie(b.Db, c)
		if len(login) > 0 {
			break
		}
	}

	// Build Post object to store
	p := goboardbackend.Post{
		Time:       goboardbackend.PostTime{Time: time.Now()},
		Login:      login,
		Info:       info,
		Message:    message,
		RawMessage: r.FormValue("message"),
	}

	// Try to store it
	if postID, err := goboardbackend.PostMessage(b.Db, p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	} else {
		w.Header().Set("X-Post-Id", strconv.FormatUint(postID, 10))
		w.WriteHeader(http.StatusNoContent)
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
		for k := range knownHeaders {
			indexes[i] = strings.Index(acceptHeader, k)
			keys[i] = k
			i++
		}

		// Find lowest non null
		minIndex := 0
		minVal := indexes[0]

		for i = 1; i < len(indexes); i++ {
			if indexes[i] >= 0 && (minVal < 0 || indexes[i] < minVal) {
				minVal = indexes[i]
				minIndex = i
			}
		}

		if minVal >= 0 { // At least one match found
			format = knownHeaders[keys[minIndex]]
		}
	}

	return
}

func postsToXML(posts []goboardbackend.Post, backendLocation string) []byte {
	var b = goboardbackend.Board{}
	if (len(backendLocation)) > 0 {
		b.Site = "http://" + backendLocation
	} else {
		b.Site = "http://localhost"
	}

	var i int
	var p goboardbackend.Post
	for i, p = range posts {
		if p.ID == 0 {
			break
		}
		posts[i].RawMessage = "" // Don't print rawData field
	}

	b.Posts = posts[:i]

	s, err := xml.Marshal(b)
	if err != nil {
		return []byte(err.Error())
	}
	return s
}

func postsToJSON(posts []goboardbackend.Post) []byte {
	var b = goboardbackend.Board{}
	b.Site = "http://localhost"

	var i int
	var p goboardbackend.Post
	for i, p = range posts {
		if p.ID == 0 {
			i--
			break
		}
		posts[i].RawMessage = "" // Don't print rawData field
	}

	b.Posts = posts[:(i + 1)]

	s, err := json.Marshal(b)
	if err != nil {
		return []byte(err.Error())
	}
	return s
}

func postsToTsv(posts []goboardbackend.Post) []byte {
	var b bytes.Buffer
	var timeText []byte

	// TSV Backend is from oldest to latests
	reverse := make([]goboardbackend.Post, len(posts))
	j := 0
	for i := len(posts) - 1; i >= 0; i-- {
		if posts[i].ID > 0 {
			reverse[j] = posts[i]
			j++
		}
	}

	for _, p := range reverse {
		if p.ID == 0 {
			break
		}

		timeText, _ = p.Time.MarshalText()
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\t%s\n",
			p.ID, timeText, p.Info, p.Login, p.Message)
	}
	return b.Bytes()
}
