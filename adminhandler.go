package main

import (
	"net/http"

	"github.com/boltdb/bolt"
)

type AdminHandler struct {
	GoboardHandler

	adminToken string
}

func NewAdminHandler(db *bolt.DB) (a *AdminHandler) {
	a = &AdminHandler{}

	a.db = db

	a.supportedOps = []SupportedOp{
		{"/admin/user", "DELETE"}, // Delete a user
		{"/admin/post", "DELETE"}, // Delete a post
	}

	a.adminToken = "plop"
	return
}

func (a *AdminHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
}
