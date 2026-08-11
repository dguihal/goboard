package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	goboardbackend "github.com/dguihal/goboard/internal/backend"
	goboarduser "github.com/dguihal/goboard/internal/user"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
)

const testToken = "supersecrettoken"

func setupTestEnv(t *testing.T) (*AdminHandler, *mux.Router, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "goboard_admin_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	f.Close()

	db, err := bbolt.Open(f.Name(), 0600, nil)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	h := NewAdminHandler(testToken)
	h.Db = db

	r := mux.NewRouter()
	for _, op := range h.SupportedOps {
		r.Handle(op.RestPath, h).Methods(op.Method)
	}

	return h, r, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

func authedRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Token-Id", testToken)
	return req
}

// --- Token auth ---

func TestAdminHandler_NoToken_Unauthorized(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin/user/alice", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAdminHandler_WrongToken_Unauthorized(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin/user/alice", nil)
	req.Header.Set("Token-Id", "wrongtoken")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// --- GET /admin/user/{login} ---

func TestGetUser_Found(t *testing.T) {
	h, router, cleanup := setupTestEnv(t)
	defer cleanup()

	if err := goboarduser.AddUser(h.Db, "alice", "password"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodGet, "/admin/user/alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var user goboarduser.User
	if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if user.Login != "alice" {
		t.Errorf("expected login 'alice', got %q", user.Login)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodGet, "/admin/user/ghost"))

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// --- DELETE /admin/user/{login} ---

func TestDeleteUser_Success(t *testing.T) {
	h, router, cleanup := setupTestEnv(t)
	defer cleanup()

	if err := goboarduser.AddUser(h.Db, "alice", "password"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodDelete, "/admin/user/alice"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// User should no longer be retrievable
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, authedRequest(http.MethodGet, "/admin/user/alice"))
	if rr2.Code != http.StatusNotFound {
		t.Errorf("expected deleted user to return 404, got %d", rr2.Code)
	}
}

// --- DELETE /admin/post/{id} ---

func TestDeletePost_Success(t *testing.T) {
	h, router, cleanup := setupTestEnv(t)
	defer cleanup()

	goboardbackend.TZLocation = time.UTC
	id, err := goboardbackend.PostMessage(h.Db, goboardbackend.Post{Message: "to delete"})
	if err != nil {
		t.Fatalf("PostMessage failed: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodDelete, "/admin/post/"+fmt.Sprintf("%d", id)))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestDeletePost_InvalidID(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodDelete, "/admin/post/not-a-number"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}


