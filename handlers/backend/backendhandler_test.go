package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	goboardbackend "github.com/dguihal/goboard/internal/backend"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
)

func setupTestEnv(t *testing.T) (*BackendHandler, *mux.Router, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "goboard_handler_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	f.Close()

	db, err := bbolt.Open(f.Name(), 0600, nil)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	goboardbackend.TZLocation = time.UTC

	h := NewBackendHandler(10, "UTC")
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

func postMessage(t *testing.T, router *mux.Router, message string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	form.Set("message", message)
	req := httptest.NewRequest(http.MethodPost, "/post", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// --- GET /backend ---

func TestGetBackend_EmptyBoard(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/backend", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestGetBackend_AfterPost_ReturnsXML(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	rr := postMessage(t, router, "hello world")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("POST /post expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/backend", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /backend expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/xml" {
		t.Errorf("expected Content-Type application/xml, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "hello world") {
		t.Errorf("expected message body to contain 'hello world', got: %s", rr.Body.String())
	}
}

// --- Format negotiation via URL suffix ---

func TestGetBackend_JSONFormat(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	postMessage(t, router, "json test")

	req := httptest.NewRequest(http.MethodGet, "/backend/json", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var board goboardbackend.Board
	if err := json.Unmarshal(rr.Body.Bytes(), &board); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(board.Posts) == 0 {
		t.Fatal("expected at least one post")
	}
}

func TestGetBackend_TSVFormat(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	postMessage(t, router, "tsv test")

	req := httptest.NewRequest(http.MethodGet, "/backend/tsv", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/tab-separated-values" {
		t.Errorf("expected TSV Content-Type, got %q", ct)
	}
}

func TestGetBackend_AcceptHeaderJSON(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	postMessage(t, router, "accept header test")

	req := httptest.NewRequest(http.MethodGet, "/backend", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json via Accept header, got %q", ct)
	}
}

// --- POST /post ---

func TestPost_EmptyMessage(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	form := url.Values{}
	form.Set("message", "")
	req := httptest.NewRequest(http.MethodPost, "/post", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty message, got %d", rr.Code)
	}
}

func TestPost_SetsXPostIdHeader(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	rr := postMessage(t, router, "track me")

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if rr.Header().Get("X-Post-Id") == "" {
		t.Error("expected X-Post-Id response header to be set")
	}
}

// --- GET /post/{id} ---

func TestGetPost_ByID(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	rr := postMessage(t, router, "fetch me")
	postID := rr.Header().Get("X-Post-Id")
	if postID == "" {
		t.Fatal("missing X-Post-Id header after post")
	}

	req := httptest.NewRequest(http.MethodGet, "/post/"+postID, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/post/99999", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetPost_InvalidID(t *testing.T) {
	_, router, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/post/not-a-number", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
