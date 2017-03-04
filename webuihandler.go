package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

// WebuiHandler represents the handler of webui URLs
type WebuiHandler struct {
	GoBoardHandler

	baseDir http.Dir
}

// NewWebuiHandler creates an WebuiHandler object
func NewWebuiHandler(webuiBaseDir string) (w *WebuiHandler) {
	w = &WebuiHandler{}

	w.baseDir = http.Dir(webuiBaseDir)

	w.supportedOps = []SupportedOp{
		{"/webui/", "/webui/", "GET", w.ServeHTTP},                // GET webui content
		{"/webui/", "/webui/{file}", "GET", w.ServeHTTP},          // GET webui file content
		{"/webui/", "/webui/{subdir}/{file}", "GET", w.ServeHTTP}, // GET webui subdir file content
	}

	return
}

func (w *WebuiHandler) ServeHTTP(wr http.ResponseWriter, rq *http.Request) {

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

	if f, err := w.baseDir.Open(filePath); err != nil {
		if os.IsNotExist(err) {
			wr.WriteHeader(http.StatusNotFound)
		} else {
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte(err.Error()))
		}
	} else {
		defer f.Close()

		if fStat, err := f.Stat(); err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			wr.Write([]byte(err.Error()))
		} else {
			http.ServeContent(wr, rq, fStat.Name(), fStat.ModTime(), f)
		}
	}
	return
}
