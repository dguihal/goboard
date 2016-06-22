package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dguihal/goboard/cookie"
	"github.com/dguihal/goboard/utils"
	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
)

type BackendHandler struct {
	GoboardHandler

	historySize int
}

type Board struct {
	XMLName xml.Name `xml:"board" json:"board"`
	Site    string   `xml:"site,attr" json:"site"`
	Posts   []Post   `xml:"" `
}

type Post struct {
	XMLName xml.Name `xml:"post"`
	Id      uint64   `xml:"id,attr" json:"id"`
	Time    PostTime `xml:"time,attr" json:"time"`
	Login   string   `xml:"login" json:"login"`
	Info    string   `xml:"info" json:"info"`
	Message string   `xml:"message" json:"message"`
}

type PostTime struct {
	time.Time
}

const PostTimeFormat = "20060102150405"

func (c PostTime) MarshalText() (result []byte, err error) {
	timS := c.Format(PostTimeFormat)
	return []byte(timS), nil
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

func (r *BackendHandler) Post(post Post) (postId uint64, err error) {
	err = r.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(postBucketName))
		if err != nil {
			return err
		}

		id, _ := b.NextSequence()
		post.Id = uint64(id)

		buf, err := json.Marshal(post)
		if err != nil {
			return err
		}

		err = b.Put(utils.IToB(post.Id), buf)

		return nil
	})

	return post.Id, err
}

func (r *BackendHandler) Get(last uint64) (posts []Post, err error) {
	r.db.View(func(tx *bolt.Tx) error {

		posts = make([]Post, r.historySize)

		b := tx.Bucket([]byte(postBucketName))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		var count int = 0

		for k, v := c.Last(); k != nil && count < r.historySize; k, v = c.Prev() {
			var p Post
			json.Unmarshal(v, &p)

			if p.Id <= last {
				break
			}
			posts[count] = p
			count++
		}

		return nil
	})
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

		// TODO : Get the session cookie and fetch the corresponding user
		cookies := rq.Cookies()

		login, err := cookie.LoginForCookie(r.db, cookies[0].Value)
		if err != nil {
			fmt.Println("POST :", err.Error())
			login = ""
		}

		p := Post{
			Time:    PostTime{time.Now()},
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

func postsToXml(posts []Post) []byte {
	var b = Board{}
	b.Site = "http://localhost"

	var index int = 0
	for _, post := range posts {
		if post.Id == 0 {
			break
		}
		index++
	}

	var pmin = make([]Post, index)
	copy(pmin, posts)
	b.Posts = pmin

	s, err := xml.Marshal(b)
	if err != nil {
		return []byte(err.Error())
	}
	return s
}

func postsToJson(posts []Post) []byte {
	return []byte("")
}

func postsToTsv(posts []Post) []byte {
	var b bytes.Buffer

	for _, post := range posts {
		if post.Id == 0 {
			break
		}
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\t%s\n",
			post.Id, post.Time.Format(PostTimeFormat), post.Info, post.Login, post.Message)
	}
	return b.Bytes()
}
