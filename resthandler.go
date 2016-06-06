package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

const postBucketName string = "Posts"

type restHandler struct {
	db *bolt.DB

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

func newRestHandler(filename string) (s *restHandler, err error) {
	s = &restHandler{}
	s.db, err = bolt.Open(filename, 0600, &bolt.Options{Timeout: 1 * time.Second})

	s.historySize = 20
	return
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func (s *restHandler) Post(post Post) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(postBucketName))
		if err != nil {
			return err
		}

		stats := b.Stats()
		fmt.Println("POST :", stats.KeyN)

		id, _ := b.NextSequence()
		post.Id = uint64(id)

		buf, err := json.Marshal(post)
		if err != nil {
			return err
		}

		err = b.Put(itob(post.Id), buf)

		return nil
	})
}

func (s *restHandler) Get(last uint64) (posts []Post, err error) {
	s.db.View(func(tx *bolt.Tx) error {

		posts = make([]Post, s.historySize)

		b := tx.Bucket([]byte(postBucketName))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		var count int = 0

		for k, v := c.Last(); k != nil && count < s.historySize; k, v = c.Prev() {
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

func (s *restHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		fmt.Println("POST")
		r.ParseForm()

		p := Post{
			Time:    PostTime{time.Now()},
			Login:   "",
			Info:    r.Header.Get("User-Agent"),
			Message: r.FormValue("message"),
		}

		err := s.Post(p)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	case "GET":
		fmt.Println("GET")

		lastAttrStr := r.URL.Query().Get("last")
		lastAttr, err := strconv.ParseUint(lastAttrStr, 10, 64)
		if err != nil {
			lastAttr = 0
		}

		fmt.Println("last :", lastAttr)

		posts, err := s.Get(lastAttr)
		if err == nil {
			vars := mux.Vars(r)
			format := vars["format"]

			var data []byte

			if format == "" || format == "xml" {
				data = postsToXml(posts)
			} else if format == "json" {
				data = postsToJson(posts)
			} else if format == "tsv" {
				data = postsToTsv(posts)
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(data))

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
