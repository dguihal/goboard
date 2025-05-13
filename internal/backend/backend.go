package backend

import (
	"encoding/json"
	"encoding/xml"
	"time"

	goboardutils "github.com/dguihal/goboard/internal/utils"
	"go.etcd.io/bbolt"
)

const backendBucketName string = "Backend"

// Post represents a user post
type Post struct {
	XMLName    xml.Name `xml:"post" json:"-"`
	ID         uint64   `xml:"id,attr" json:"id"`
	Time       PostTime `xml:"time,attr" json:"time"`
	Login      string   `xml:"login" json:"login"`
	Info       string   `xml:"info" json:"info"`
	Message    string   `xml:"message" json:"message"`
	RawMessage string   `xml:"-" json:"rawmessage,omitempty"`
}

// TZLocation is destination TZ location for backends in TSV and XML
var TZLocation *time.Location

// PostTime represents the timestamp of a user post
type PostTime struct {
	time.Time
}

// PostTimeFormat is the format used to convert a PostTime to a byte array
const PostTimeFormat = "20060102150405"

// MarshalText converts a PostTime to a byte array
func (pt PostTime) MarshalText() (result []byte, err error) {
	timeS := pt.In(TZLocation).Format(PostTimeFormat)
	return []byte(timeS), nil
}

// Board represents the base struture for a board backend
type Board struct {
	XMLName xml.Name `xml:"board" json:"-"`
	Site    string   `xml:"site,attr" json:"site"`
	Posts   []Post   `xml:"" `
}

// DeletePost is a method for deleting a post from the history
func DeletePost(db *bbolt.DB, id uint64) (err error) {

	err = db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(backendBucketName))
		if err != nil {
			return err
		}

		return b.Delete(goboardutils.IToB(id))
	})
	return
}

// GetBackend returns the last posts from the history
func GetBackend(db *bbolt.DB, historySize int, last uint64) (posts []Post, err error) {

	posts = make([]Post, historySize)

	err = db.View(func(tx *bbolt.Tx) error {

		b := tx.Bucket([]byte(backendBucketName))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		var count int

		for k, v := c.Last(); k != nil && count < historySize; k, v = c.Prev() {
			var p Post
			err = json.Unmarshal(v, &p)
			if err != nil {
				return err
			}

			if p.ID <= last {
				break
			}
			posts[count] = p
			count++
		}

		return nil
	})
	return
}

// GetPost returns a post from its id
func GetPost(db *bbolt.DB, id uint64) (post Post, err error) {

	post = Post{}

	err = db.View(func(tx *bbolt.Tx) error {

		b := tx.Bucket([]byte(backendBucketName))
		if b == nil {
			return nil
		}

		v := b.Get(goboardutils.IToB(id))
		if v != nil {
			err := json.Unmarshal(v, &post)
			if err != nil {
				return err
			}
		}

		return nil
	})
	return
}

// PostMessage adds a new message to the history
func PostMessage(db *bbolt.DB, post Post) (postID uint64, err error) {

	err = db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(backendBucketName))
		if err != nil {
			return err
		}

		id, err := b.NextSequence()
		if err != nil {
			return err
		}

		postID = uint64(id)
		post.ID = postID

		buf, err := json.Marshal(post)
		if err != nil {
			return err
		}

		_ = b.Put(goboardutils.IToB(post.ID), buf)

		return nil
	})

	return
}
