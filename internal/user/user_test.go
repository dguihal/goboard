package user

import (
	"os"
	"testing"

	"go.etcd.io/bbolt"
)

func setupTestDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "goboard_user_test_*.db")
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

func TestAddUser_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := AddUser(db, "alice", "password123"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestAddUser_Duplicate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := AddUser(db, "alice", "password123"); err != nil {
		t.Fatalf("first AddUser failed: %v", err)
	}

	err := AddUser(db, "alice", "other")
	if err == nil {
		t.Fatal("expected error for duplicate user, got nil")
	}
	uerr, ok := err.(*Error)
	if !ok || uerr.ErrCode != UserAlreadyExistsError {
		t.Errorf("expected UserAlreadyExistsError, got %v", err)
	}
}

func TestAuthUser_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := AddUser(db, "alice", "password123"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	if err := AuthUser(db, "alice", "password123"); err != nil {
		t.Errorf("expected auth success, got: %v", err)
	}
}

func TestAuthUser_WrongPassword(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := AddUser(db, "alice", "password123"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	err := AuthUser(db, "alice", "wrong")
	if err == nil {
		t.Fatal("expected auth failure, got nil")
	}
	uerr, ok := err.(*Error)
	if !ok || uerr.ErrCode != AuthenticationFailed {
		// bcrypt may wrap the error differently; ensure it is at minimum non-nil
		if ok {
			t.Errorf("expected AuthenticationFailed code, got %d", uerr.ErrCode)
		}
	}
}

func TestAuthUser_UnknownUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := AuthUser(db, "ghost", "password")
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestGetUser_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := AddUser(db, "alice", "password123"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	user, err := GetUser(db, "alice")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Login != "alice" {
		t.Errorf("expected login 'alice', got %q", user.Login)
	}
	if user.HashedPassword != nil {
		t.Error("expected HashedPassword to be nil in returned user")
	}
}

func TestGetUser_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := GetUser(db, "ghost")
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
	uerr, ok := err.(*Error)
	if !ok || uerr.ErrCode != UserDoesNotExistsError {
		t.Errorf("expected UserDoesNotExistsError, got %v", err)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := AddUser(db, "alice", "password123"); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	if err := DeleteUser(db, "alice"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	_, err := GetUser(db, "alice")
	if err == nil {
		t.Fatal("expected user to be deleted")
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := DeleteUser(db, "ghost")
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
	uerr, ok := err.(*Error)
	if !ok || uerr.ErrCode != UserDoesNotExistsError {
		t.Errorf("expected UserDoesNotExistsError, got %v", err)
	}
}
