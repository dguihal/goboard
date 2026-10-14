package cookie

import (
	"net/http"
	"os"
	"testing"
	"time"

	goboarduser "github.com/dguihal/goboard/internal/user"
	"go.etcd.io/bbolt"
)

func setupTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "goboard_cookie_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	f.Close()

	db, err := bbolt.Open(f.Name(), 0600, nil)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

func TestForUser_CreatesCookie(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cookie, err := ForUser(db, "alice", 7)
	if err != nil {
		t.Fatalf("ForUser failed: %v", err)
	}
	if cookie.Name != goboardCookieName {
		t.Errorf("expected cookie name %q, got %q", goboardCookieName, cookie.Name)
	}
	if len(cookie.Value) == 0 {
		t.Error("expected non-empty cookie value")
	}
	if cookie.Expires.Before(time.Now()) {
		t.Error("expected cookie expiry to be in the future")
	}
}

func TestForUser_ReturnsExistingCookie(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	c1, err := ForUser(db, "alice", 7)
	if err != nil {
		t.Fatalf("first ForUser failed: %v", err)
	}

	c2, err := ForUser(db, "alice", 7)
	if err != nil {
		t.Fatalf("second ForUser failed: %v", err)
	}

	if c1.Value != c2.Value {
		t.Error("expected same cookie returned for existing user, got different values")
	}
}

func TestLoginForCookie_Valid(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := goboarduser.AddUser(db, "alice", "pass"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	stored, err := ForUser(db, "alice", 7)
	if err != nil {
		t.Fatalf("ForUser failed: %v", err)
	}

	login, err := LoginForCookie(db, &stored)
	if err != nil {
		t.Fatalf("LoginForCookie failed: %v", err)
	}
	if login != "alice" {
		t.Errorf("expected login 'alice', got %q", login)
	}
}

func TestLoginForCookie_WrongName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cookie := &http.Cookie{
		Name:    "wrong_name",
		Value:   "somevalue",
		Expires: time.Now().Add(24 * time.Hour),
	}

	login, _ := LoginForCookie(db, cookie)
	if login != "" {
		t.Errorf("expected empty login for wrong cookie name, got %q", login)
	}
}

func TestLoginForCookie_Expired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cookie := &http.Cookie{
		Name:    goboardCookieName,
		Value:   "somevalue",
		Expires: time.Now().Add(-24 * time.Hour), // expired
	}

	login, _ := LoginForCookie(db, cookie)
	if login != "" {
		t.Errorf("expected empty login for expired cookie, got %q", login)
	}
}

func TestDeleteCookiesForUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	stored, err := ForUser(db, "alice", 7)
	if err != nil {
		t.Fatalf("ForUser failed: %v", err)
	}

	if err := DeleteCookiesForUser(db, "alice"); err != nil {
		t.Fatalf("DeleteCookiesForUser failed: %v", err)
	}

	login, _ := LoginForCookie(db, &stored)
	if login != "" {
		t.Errorf("expected empty login after cookie deletion, got %q", login)
	}
}
