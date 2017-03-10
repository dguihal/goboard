package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

// Config holds the configuration of the process
type Config struct {
	ListenPort        string      `yaml:"ListenPort"`
	BasePath          string      `yaml:"BasePath"`
	BackendTimeZone   string      `yaml:"BackendTimeZone"`
	MaxHistorySize    int         `yaml:"MaxHistorySize"`
	CookieDuration    int         `yaml:"CookieDuration"`
	GoBoardDBFile     string      `yaml:"GoBoardDBFile"`
	GoBoardDBFileMode os.FileMode `yaml:"GoBoardDBFileMode"`
	AccessLogFile     string      `yaml:"AccessLogFile"`
	AccessLogFileMode os.FileMode `yaml:"AccessLogFileMode"`
	SwaggerPath       string      `yaml:"SwaggerPath"`
	WebuiPath         string      `yaml:"WebuiPath"`
	AdminToken        string      `yaml:"AdminToken"`
}

// RESTEndpointHandler defines a handler function for a REST Endpoint
type RESTEndpointHandler func(http.ResponseWriter, *http.Request)

// SupportedOp Defines a REST endpoint with its path, method and endpoint
type SupportedOp struct {
	PathBase string
	RestPath string
	Method   string
	handler  RESTEndpointHandler
}

// GoBoardHandler Base Class for endpoint handlers
type GoBoardHandler struct {
	Db           *bolt.DB
	supportedOps []SupportedOp
	BasePath     string
}

func main() {
	configData, err := ioutil.ReadFile("goboard.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Manage access log file
	if config.AccessLogFileMode == 0 {
		config.AccessLogFileMode = 0660
	}
	var fiAccessLog *os.File
	if len(config.AccessLogFile) > 0 {
		fiAccessLog, err = os.OpenFile(config.AccessLogFile, os.O_RDWR|os.O_APPEND, config.AccessLogFileMode)
		if err != nil {
			fiAccessLog, err = os.Create(config.AccessLogFile)
			if err != nil {
				log.Fatalf("error: %v", err)
			}
		}
	}

	if config.GoBoardDBFileMode == 0 {
		config.GoBoardDBFileMode = 0600
	}
	db, err := bolt.Open(config.GoBoardDBFile, config.GoBoardDBFileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	mainRouter := mux.NewRouter().StrictSlash(true)
	r := mainRouter
	if len(config.BasePath) > 0 {
		r = mainRouter.PathPrefix(config.BasePath).Subrouter()
	}
	//router.HandleFunc("/", Index)

	// Backend operations
	backendHandler := NewBackendHandler(config.MaxHistorySize, config.BackendTimeZone)
	backendHandler.BasePath = config.BasePath
	backendHandler.Db = db
	for _, op := range backendHandler.supportedOps {
		r.Handle(op.RestPath, backendHandler).Methods(op.Method)
	}

	// User operations
	userHandler := NewUserHandler(config.CookieDuration)
	userHandler.BasePath = config.BasePath
	userHandler.Db = db
	for _, op := range userHandler.supportedOps {
		r.Handle(op.RestPath, userHandler).Methods(op.Method)
	}

	// Admin operations
	adminHandler := NewAdminHandler(config.AdminToken)
	adminHandler.BasePath = config.BasePath
	adminHandler.Db = db
	for _, op := range adminHandler.supportedOps {
		r.Handle(op.RestPath, adminHandler).Methods(op.Method)
	}

	// Swagger operations
	if len(config.SwaggerPath) > 0 {
		realPath := os.ExpandEnv(config.SwaggerPath)
		if _, err := os.Stat(strings.Join([]string{realPath, "/index.html"}, "")); os.IsNotExist(err) {
			log.Println(strings.Join([]string{realPath, "/index.html"}, ""), "Not found: Disabling swagger capabilities")
		} else {
			swaggerHandler := NewSwaggerHandler(realPath)
			swaggerHandler.BasePath = config.BasePath
			for _, op := range swaggerHandler.supportedOps {
				r.Handle(op.RestPath, swaggerHandler).Methods(op.Method)
			}
		}
	}

	// Webui operations
	if len(config.WebuiPath) > 0 {
		realPath := os.ExpandEnv(config.WebuiPath)
		if _, err := os.Stat(strings.Join([]string{realPath, "/index.html"}, "")); os.IsNotExist(err) {
			log.Println(strings.Join([]string{realPath, "/index.html"}, ""), "Not found: Disabling webui capabilities")
		} else {
			webuiHandler := NewWebuiHandler(realPath)
			webuiHandler.BasePath = config.BasePath
			for _, op := range webuiHandler.supportedOps {
				r.Handle(op.RestPath, webuiHandler).Methods(op.Method)
			}
		}
	}

	fmt.Println("GoBoard version 0.0.1 starting on port", config.ListenPort)

	var handler http.Handler
	if fiAccessLog != nil {
		handler = handlers.LoggingHandler(fiAccessLog, mainRouter)
	} else {
		handler = mainRouter
	}
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", config.ListenPort), handler))
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

//http://thenewstack.io/make-a-restful-json-api-go/
//http://stevenwhite.com/building-a-rest-service-with-golang-1/
//http://stevenwhite.com/building-a-rest-service-with-golang-2/
//http://stevenwhite.com/building-a-rest-service-with-golang-3/
//https://github.com/golang/go/wiki/LearnServerProgramming
//https://astaxie.gitbooks.io/build-web-application-with-golang/

//https://github.com/boltdb/bolt : Backend
//https://github.com/skyec/boltdb-server/blob/master/server.go

//https://blog.golang.org/error-handling-and-go
