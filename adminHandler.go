package main

import (
	"net/http"

	"github.com/boltdb/bolt"
)

type adminHandler struct {
	GoboardHandler

	adminToken string
}

func newAdminHandler(db *bolt.DB) (a *adminHandler) {
	a = &adminHandler{}

	a.db = db

	a.supportedOps = []SupportedOp{
		{"/admin/user", "DELETE"}, // Delete a user
		{"/admin/post", "DELETE"}, // Delete a post
	}

	a.adminToken = "plop"
	return
}

func (a *adminHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
}
