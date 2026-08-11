package user

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"go.etcd.io/bbolt"
)

type UserTestEnv struct {
	DB *bbolt.DB
}

// setupTestEnv create temporary test env
func setupTestEnv(t *testing.T) (*UserTestEnv, func()) {
	tmpFile := "test_auth.db"
	db, err := bbolt.Open(tmpFile, 0600, nil)
	if err != nil {
		t.Fatalf("Erreur d'ouverture BoltDB : %v", err)
	}

	// Initialise les buckets nécessaires
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		return err
	})
	if err != nil {
		t.Fatalf("Erreur d'init bucket : %v", err)
	}

	env := &UserTestEnv{DB: db}

	// Fonction de nettoyage
	cleanup := func() {
		db.Close()
		os.Remove(tmpFile)
	}

	return env, cleanup
}

func TestUserHandlerWhoami403(t *testing.T) {

	env, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/user/whoami", nil)

	// Response recorder to capture handler's response
	rr := httptest.NewRecorder()

	userHandler := NewUserHandler(30)
	userHandler.Db = env.DB

	userHandler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("Wrong response status code: got %v, expected %v", status, http.StatusForbidden)
	}

	expected := "You need to be authenticated"
	if rr.Body.String() != expected {
		t.Errorf("Wrong response body: got %v, expected %v", rr.Body.String(), expected)
	}
}

func TestUserHandlerRegisterLogin(t *testing.T) {

	env, cleanup := setupTestEnv(t)
	defer cleanup()

	// Step 1: Create user
	form := url.Values{}
	form.Add("login", "alice")
	form.Add("password", "secure123")

	userAddReq := httptest.NewRequest(http.MethodPost, "/user/add", strings.NewReader(form.Encode()))
	userAddReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	userAddRes := httptest.NewRecorder()

	userHandler := NewUserHandler(30)
	userHandler.Db = env.DB

	userHandler.ServeHTTP(userAddRes, userAddReq)

	if userAddRes.Code != http.StatusCreated {
		t.Fatalf("User creation failed: %d - %s", userAddRes.Code, userAddRes.Body.String())
	}

	// Step 2: Authenticate user
	loginReq := httptest.NewRequest(http.MethodPost, "/user/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRes := httptest.NewRecorder()

	userHandler.ServeHTTP(loginRes, loginReq)

	if loginRes.Code != http.StatusOK {
		t.Fatalf("Authentification failed: %d - %s", loginRes.Code, loginRes.Body.String())
	}

	// Check cookie in response
	cookies := loginRes.Result().Cookies()
	found := false

	if len(cookies) == 0 {
		t.Fatalf("No cookies returned")
	}

	for _, cookie := range cookies {
		if cookie.Name == "goboard_id" {
			found = true
			if cookie.Value == "" {
				t.Errorf("Cookie 'goboard_id' present but empty")
			}
			break
		}
	}

	if !found {
		t.Errorf("Cookie 'goboard_id' not found")
	}

}
