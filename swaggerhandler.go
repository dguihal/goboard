package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

// SwaggerHandler represents the handler of swagger URLs
type SwaggerHandler struct {
	GoBoardHandler

	baseDir http.Dir
}

// NewSwaggerHandler creates an SwaggerHandler object
func NewSwaggerHandler(swaggerBaseDir string) (s *SwaggerHandler) {
	s = &SwaggerHandler{}

	s.baseDir = http.Dir(swaggerBaseDir)

	s.supportedOps = []SupportedOp{
		{"/swagger/", "/swagger/", "GET", s.ServeHTTP},                // GET swagger content
		{"/swagger/", "/swagger/{file}", "GET", s.ServeHTTP},          // GET swagger file content
		{"/swagger/", "/swagger/{subdir}/{file}", "GET", s.ServeHTTP}, // GET swagger subdir file content
	}
	return
}

func (s *SwaggerHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {

	vars := mux.Vars(rq)
	filePath := vars["file"]
	subDirPath := vars["subdir"]
	if len(subDirPath) > 0 {
		filePath = subDirPath + "/" + filePath
	}

	if len(filePath) == 0 || strings.HasSuffix(filePath, "/") {
		filePath = filePath + "index.html"
	}

	fmt.Println(filePath)

	if f, err := s.baseDir.Open(filePath); err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	} else {
		defer f.Close()

		if fStat, err := f.Stat(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			http.ServeContent(w, rq, fStat.Name(), fStat.ModTime(), f)
		}
	}
	return
}
