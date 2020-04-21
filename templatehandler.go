package main

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
)

// TemplateHandler represents the handler of swagger URLs
type TemplateHandler struct {
	swaggerBaseDir    http.Dir
	swaggerBaseDirSet bool
	webuiBaseDir      http.Dir
	webuiBaseDirSet   bool

	BasePath string
}

// NewTemplateHandler creates an TemplateHandler object
func NewTemplateHandler() *TemplateHandler {
	s := new(TemplateHandler)

	s.swaggerBaseDirSet = false
	s.webuiBaseDirSet = false

	return s
}

// SetSwaggerBaseDir configures swagger base dir
func (s *TemplateHandler) SetSwaggerBaseDir(swaggerBaseDir string) {
	s.swaggerBaseDir = http.Dir(swaggerBaseDir)

	s.swaggerBaseDirSet = true
}

// GetSwaggerOp GET swagger content
func (s *TemplateHandler) GetSwaggerOp() SupportedOp {
	return SupportedOp{"/swagger/swagger.yaml", "/swagger/swagger.yaml", "GET", s.ServeSwagger}
}

// SetSwaggerBaseDir configures webui base dir
func (s *TemplateHandler) setWebUIBaseDir(webuiBaseDir string) {
	s.webuiBaseDir = http.Dir(webuiBaseDir)

	s.webuiBaseDirSet = true
}

// GetWebUIOp GET WebUI config content
func (s *TemplateHandler) GetWebUIOp() SupportedOp {
	return SupportedOp{"/webui/js/settings.js", "/webui/js/settings.js", "GET", s.ServeConfigJS}
}

// ServeConfigJS handles webui config.js file
func (s *TemplateHandler) ServeConfigJS(w http.ResponseWriter, rq *http.Request) {
	if !s.webuiBaseDirSet {
		w.WriteHeader(http.StatusNotFound)
	} else if f, err := s.webuiBaseDir.Open("js/settings.js"); err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	} else {
		defer f.Close()

		tmpl := template.New("ConfigJS")

		// Read swagger template data
		buf := new(bytes.Buffer)
		buf.ReadFrom(f)
		str := buf.String()

		data := struct {
			BasePath string
		}{
			map[bool]string{true: s.BasePath, false: "/"}[len(s.BasePath) > 0],
		}

		tmpl, _ = tmpl.Parse(str) // Parse template file.
		tmpl.Execute(w, data)
	}
}

// ServeSwagger handles swagger yaml data file
func (s *TemplateHandler) ServeSwagger(w http.ResponseWriter, rq *http.Request) {

	if !s.swaggerBaseDirSet {
		w.WriteHeader(http.StatusNotFound)
	} else if f, err := s.swaggerBaseDir.Open("swagger.yaml"); err != nil {
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
