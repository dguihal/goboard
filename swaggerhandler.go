package main

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
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
		{"/swagger/swagger.yaml", "/swagger/swagger.yaml", "GET", s.ServeHTTP}, // GET swagger content
	}
	return
}

func (s *SwaggerHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {

	if f, err := s.baseDir.Open("swagger.yaml"); err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	} else {
		defer f.Close()

		tmpl := template.New("Swagger")

		// Read swagger template data
		buf := new(bytes.Buffer)
		buf.ReadFrom(f)
		str := buf.String()

		// Try to guess original Scheme
		scheme := rq.URL.Scheme
		if len(rq.Header.Get("X-Forwarded-Proto")) > 0 {
			scheme = rq.Header.Get("X-Forwarded-Proto")
		}
		if len(scheme) == 0 {
			scheme = "http" // Default value
		}

		// Try to guess original Host
		host := rq.Host
		if len(rq.Header.Get("X-Forwarded-Host")) > 0 {
			host = rq.Header.Get("X-Forwarded-Host")
		}

		data := struct {
			Scheme   string
			Hostname string
			BasePath string
		}{
			scheme,
			host,
			map[bool]string{true: s.BasePath, false: "/"}[len(s.BasePath) > 0],
		}

		tmpl, _ = tmpl.Parse(str) // Parse template file.
		tmpl.Execute(w, data)
	}
	return
}
