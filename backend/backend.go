// user.go
package user

import (
	"encoding/json"
	"encoding/xml"
	"time"

	"github.com/boltdb/bolt"
	goboardutils "github.com/dguihal/goboard/utils"
)

const postBucketName string = "Posts"

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

func (c PostTime) MarshalText() (result []byte, err error) {
	timeS := c.Format(PostTimeFormat)
	return []byte(timeS), nil
}

const PostTimeFormat = "20060102150405"

type Board struct {
	XMLName xml.Name `xml:"board" json:"board"`
	Site    string   `xml:"site,attr" json:"site"`
	Posts   []Post   `xml:"" `
}

func DeletePost(db *bolt.DB, id uint64) (err error) {

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(postBucketName))
		if err != nil {
			return err
		}

		return b.Delete(goboardutils.IToB(id))
	})
	return
}

func GetBackend(db *bolt.DB, historySize int, last uint64) (posts []Post, err error) {

	posts = make([]Post, historySize)

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(postBucketName))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		var count int = 0

		for k, v := c.Last(); k != nil && count < historySize; k, v = c.Prev() {
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

func PostMessage(db *bolt.DB, post Post) (postId uint64, err error) {

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(postBucketName))
		if err != nil {
			return err
		}

		id, _ := b.NextSequence()
		postId = uint64(id)

		buf, err := json.Marshal(post)
		if err != nil {
			return err
		}

		err = b.Put(goboardutils.IToB(post.Id), buf)

		return nil
	})

	return
}
