package handlers

import (
	"net/http"

	"go.etcd.io/bbolt"
)

// RESTEndpointHandler defines a handler function for a REST Endpoint
type RESTEndpointHandler func(http.ResponseWriter, *http.Request)

// SupportedOp Defines a REST endpoint with its path, method and endpoint
type SupportedOp struct {
	PathBase string
	RestPath string
	Method   string
	Handler  RESTEndpointHandler
}

// GoBoardHandler Base Class for endpoint handlers
type GoBoardHandler struct {
	Db           *bbolt.DB
	SupportedOps []SupportedOp
}
