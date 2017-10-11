package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	goboardbackend "github.com/dguihal/goboard/backend"
	goboardcookie "github.com/dguihal/goboard/cookie"
	"github.com/gorilla/mux"
	"github.com/hishboy/gocommons/lang"
)

const defaultFormat string = "xml"

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
	tzLocation  time.Location
}

func getOffsetFromLocation(tz string) (TZShift int) {
	offset := 0
	if location, err := time.LoadLocation(tz); err == nil {
		t := time.Now()
		_, offsetBase := t.Zone()                //TZ of process
		_, offsetTarget := t.In(location).Zone() //TZ configured
		offset = offsetTarget - offsetBase
		fmt.Println(offsetTarget, offsetBase, offsetTarget-offsetBase)
	} else {
		fmt.Println("Error : ", err, ". Falling back to UTC")
	}
	return offset
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
	b.BasePath = ""
	return
}

func (b *BackendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)

	for _, op := range b.supportedOps {
		if r.Method == op.Method && strings.HasPrefix(r.URL.Path, b.BasePath+op.PathBase) {
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

	return
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

	return
}

func (b *BackendHandler) post(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	message := sanitize(r.FormValue("message"))
	rawInfo := r.FormValue("info")
	if len(rawInfo) == 0 {
		rawInfo = r.Header.Get("User-Agent")
	}
	info := sanitize(rawInfo)
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

	for _, p := range posts {
		if p.ID == 0 {
			break
		}

		timeText, _ = p.Time.MarshalText()
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\t%s\n",
			p.ID, timeText, p.Info, p.Login, p.Message)
	}
	return b.Bytes()
}

/******************************************************************
 *             Backend Sanitizer
 ******************************************************************/

// Sanitizer entry point
func sanitize(input string) string {

	if len(input) == 0 {
		return ""
	}

	tmp := stripCtlFromUTF8(input)
	tmp = htmlEscape(tmp)
	return tmp
}

// Remove unwanted (control) characters
func stripCtlFromUTF8(str string) string {
	return strings.Map(func(r rune) rune {
		if r >= 32 && r != 127 {
			return r
		}
		return -1
	}, str)
}

// HTML escape some conflicting characters
func sanitizeChars(input string) string {
	tmp := strings.Replace(input, "&", "&amp;", -1)
	tmp = strings.Replace(tmp, "<", "&lt;", -1)
	return strings.Replace(tmp, ">", "&gt;", -1)
}

// Allowed tags dictionnary
var allowedTags = map[string]bool{
	"a":  true,
	"b":  true,
	"i":  true,
	"s":  true,
	"tt": true,
	"em": true,
	"u":  true,
}

// Allowed attributes for tag dictionnary
var allowedAttrForTags = map[string][]string{
	"a": []string{"href"},
}

type token struct {
	txt       string
	tagName   string
	tokenType html.TokenType
}

func htmlEscape(input string) string {

	s := lang.NewStack()
	tagCount := map[string]int{}

	z := html.NewTokenizer(strings.NewReader(input))

L:
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			break L
		case tt == html.StartTagToken:
			tn, hasAttrs := z.TagName()
			tnStr := string(tn)

			// Tag belongs to allowed list
			if allowedTags[tnStr] {
				tagAttrsStr := ""

				// Tag attributes management
				if allowedAttrs := allowedAttrForTags[tnStr]; hasAttrs && allowedAttrs != nil {

					moreAttr := hasAttrs
					for moreAttr {
						var key, val []byte
						key, val, moreAttr = z.TagAttr()
						for _, allowedAttr := range allowedAttrs {
							if string(key) == allowedAttr {
								tagAttrsStr = fmt.Sprintf(" %s=\"%s\"", string(key), val)
							}
						}
					}
				}

				s.Push(token{
					txt:       fmt.Sprintf("<%s%s>", tn, tagAttrsStr),
					tagName:   tnStr,
					tokenType: html.StartTagToken})

				// if a key doesn't exists it's value is 0
				tagCount[tnStr] = tagCount[tnStr] + 1
			} else {
				s.Push(token{
					txt:       sanitizeChars(string(z.Raw())),
					tokenType: html.TextToken})
			}
		case tt == html.EndTagToken:
			tn, _ := z.TagName()
			tnStr := string(tn)

			if allowedTags[tnStr] && tagCount[tnStr] > 0 {
				endStr := fmt.Sprintf("</%s>", tn)

				str := ""
				startTagFound := false
				for s.Len() > 0 {
					tmp := s.Pop().(token)

					if tmp.tokenType == html.StartTagToken && tmp.tagName != tnStr {
						// Not a corresponding open tag : sanitize it and store it as text
						str = fmt.Sprintf("%s%s", sanitizeChars(tmp.txt), str)
					} else {
						// a text or a corresponding open tag, at it as is
						str = fmt.Sprintf("%s%s", tmp.txt, str)

						if tmp.tagName == tnStr {
							startTagFound = true
							str = fmt.Sprintf("%s%s", str, endStr)
							// and leave if it's a corresponding open tag
							break
						}
					}
				}
				if !startTagFound {
					str = fmt.Sprintf("%s%s", str, sanitizeChars(endStr))
				}

				s.Push(token{
					txt:       str,
					tokenType: html.TextToken})
			} else {
				s.Push(token{
					txt:       sanitizeChars(string(z.Raw())),
					tokenType: html.TextToken})
			}

		default:
			re := regexp.MustCompile("(?i)https?://[\\da-z\\.-]+(?::\\d+)?(?:/[^\\s\"]*)*/?")
			raw := string(z.Raw())
			if matches := re.FindAllStringIndex(raw, -1); matches != nil {
				start := 0
				for _, match := range matches {
					if start < match[0] {
						s.Push(token{
							txt:       sanitizeChars(raw[start:match[0]]),
							tokenType: html.TextToken})
					}
					var buffer bytes.Buffer
					buffer.WriteString("<a href=\"")
					buffer.WriteString(raw[match[0]:match[1]])
					buffer.WriteString("\">[url]</a>")

					s.Push(token{
						txt:       buffer.String(),
						tokenType: html.TextToken})

					start = match[1]
				}

				if start < (len(raw)) {
					s.Push(token{
						txt:       sanitizeChars(raw[start:len(raw)]),
						tokenType: html.TextToken})

				}
			} else {
				s.Push(token{
					txt:       sanitizeChars(raw),
					tokenType: html.TextToken})
			}
		}
	}

	str := ""
	for s.Len() > 0 {
		tmp := s.Pop().(token)

		if tmp.tokenType != html.TextToken {
			str = fmt.Sprintf("%s%s", sanitizeChars(tmp.txt), str)
		} else {
			str = fmt.Sprintf("%s%s", tmp.txt, str)
		}
	}
	return str
}
