// user.go
package user

import (
	"encoding/json"
	"encoding/xml"
	"time"

	"github.com/boltdb/bolt"
	goboardutils "github.com/dguihal/goboard/utils"
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

// PostTime represents the timestamp of a user post
type PostTime struct {
	time.Time
}

// PostTimeFormat is the format used to convert a PostTime to a byte array
const PostTimeFormat = "20060102150405"

// MarshalText converts a PostTime to a byte array
func (pt PostTime) MarshalText() (result []byte, err error) {
	_, offset := pt.Zone()
	timeS := pt.Add(time.Duration(offset) * time.Second).Format(PostTimeFormat)
	return []byte(timeS), nil
}

// Board represents the base struture for a board backend
type Board struct {
	XMLName xml.Name `xml:"board" json:"-"`
	Site    string   `xml:"site,attr" json:"site"`
	Posts   []Post   `xml:"" `
}

// DeletePost is a method for deleting a post from the history
func DeletePost(db *bolt.DB, id uint64) (err error) {

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(backendBucketName))
		if err != nil {
			return err
		}

		return b.Delete(goboardutils.IToB(id))
	})
	return
}

// GetBackend returns the last posts from the history
func GetBackend(db *bolt.DB, historySize int, last uint64) (posts []Post, err error) {

	posts = make([]Post, historySize)

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(backendBucketName))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		var count int

		for k, v := c.Last(); k != nil && count < historySize; k, v = c.Prev() {
			var p Post
			json.Unmarshal(v, &p)

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
func GetPost(db *bolt.DB, id uint64) (post Post, err error) {

	post = Post{}

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(backendBucketName))
		if b == nil {
			return nil
		}

		v := b.Get(goboardutils.IToB(id))
		if v != nil {
			json.Unmarshal(v, &post)
		}

		return nil
	})
	return
}

// PostMessage adds a new message to the history
func PostMessage(db *bolt.DB, post Post) (postID uint64, err error) {

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(backendBucketName))
		if err != nil {
			return err
		}

		id, _ := b.NextSequence()
		postID = uint64(id)
		post.ID = postID

		buf, err := json.Marshal(post)
		if err != nil {
			return err
		}

		err = b.Put(goboardutils.IToB(post.ID), buf)

		return nil
	})

	return
}
