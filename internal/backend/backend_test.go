package backend

import (
	"os"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func setupTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "goboard_backend_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	f.Close()

	db, err := bbolt.Open(f.Name(), 0600, nil)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	TZLocation = time.UTC

	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

func newPost(message string) Post {
	return Post{
		Time:       PostTime{Time: time.Now()},
		Login:      "alice",
		Info:       "test-agent",
		Message:    message,
		RawMessage: message,
	}
}

func TestPostMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	id, err := PostMessage(db, newPost("hello"))
	if err != nil {
		t.Fatalf("PostMessage error: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero post ID")
	}
}

func TestGetBackend_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	posts, err := GetBackend(db, 10, 0)
	if err != nil {
		t.Fatalf("GetBackend error: %v", err)
	}
	for _, p := range posts {
		if p.ID != 0 {
			t.Fatalf("expected empty board, got post ID %d", p.ID)
		}
	}
}

func TestGetBackend_ReturnsPostedMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	for _, msg := range []string{"first", "second", "third"} {
		if _, err := PostMessage(db, newPost(msg)); err != nil {
			t.Fatalf("PostMessage error: %v", err)
		}
	}

	posts, err := GetBackend(db, 10, 0)
	if err != nil {
		t.Fatalf("GetBackend error: %v", err)
	}

	count := 0
	for _, p := range posts {
		if p.ID != 0 {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 posts, got %d", count)
	}
}

func TestGetBackend_HistorySizeLimits(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		if _, err := PostMessage(db, newPost("msg")); err != nil {
			t.Fatalf("PostMessage error: %v", err)
		}
	}

	posts, err := GetBackend(db, 3, 0)
	if err != nil {
		t.Fatalf("GetBackend error: %v", err)
	}

	count := 0
	for _, p := range posts {
		if p.ID != 0 {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 posts (historySize), got %d", count)
	}
}

func TestGetBackend_AfterLastID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	id1, _ := PostMessage(db, newPost("first"))
	_, _ = PostMessage(db, newPost("second"))

	posts, err := GetBackend(db, 10, id1)
	if err != nil {
		t.Fatalf("GetBackend error: %v", err)
	}

	for _, p := range posts {
		if p.ID != 0 && p.ID <= id1 {
			t.Fatalf("expected only posts after ID %d, got ID %d", id1, p.ID)
		}
	}
}

func TestGetPost(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	id, err := PostMessage(db, newPost("hello"))
	if err != nil {
		t.Fatalf("PostMessage error: %v", err)
	}

	post, err := GetPost(db, id)
	if err != nil {
		t.Fatalf("GetPost error: %v", err)
	}
	if post.ID != id {
		t.Fatalf("expected post ID %d, got %d", id, post.ID)
	}
	if post.Message != "hello" {
		t.Fatalf("expected message 'hello', got %q", post.Message)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	post, err := GetPost(db, 99999)
	if err != nil {
		t.Fatalf("GetPost error: %v", err)
	}
	if post.ID != 0 {
		t.Fatalf("expected zero post for unknown ID, got %d", post.ID)
	}
}

func TestDeletePost(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	id, err := PostMessage(db, newPost("to delete"))
	if err != nil {
		t.Fatalf("PostMessage error: %v", err)
	}

	if err := DeletePost(db, id); err != nil {
		t.Fatalf("DeletePost error: %v", err)
	}

	post, err := GetPost(db, id)
	if err != nil {
		t.Fatalf("GetPost after delete error: %v", err)
	}
	if post.ID != 0 {
		t.Fatalf("expected deleted post to be gone, got ID %d", post.ID)
	}
}
