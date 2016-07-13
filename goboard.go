package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ListenPort     string `yaml:"ListenPort"`
	MaxHistorySize int    `yaml:"MaxHistorySize"`
	CookieDuration int    `yaml:"CookieDuration"`
	AccessLogFile  string `yaml:"AccessLogFile"`
}

type SupportedOp struct {
	path   string
	method string
}

type GoboardHandler struct {
	db           *bolt.DB
	supportedOps []SupportedOp
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
	var fiAccessLog *os.File = nil
	if len(config.AccessLogFile) > 0 {
		fiAccessLog, err = os.OpenFile(config.AccessLogFile, os.O_RDWR|os.O_APPEND, 0666);
		if err != nil {
			fiAccessLog, err = os.Create(config.AccessLogFile)
			if err != nil {
				log.Fatalf("error: %v", err)
			}
		}
	}

	db, err := bolt.Open("my.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	muxRouter := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", Index)c

	// Backend operations
	backendHandler := NewBackendHandler(db, config.MaxHistorySize)
	for _, op := range backendHandler.supportedOps {
		muxRouter.Handle(op.path, backendHandler).Methods(op.method)
	}

	// User operations
	userHandler := NewUserHandler(db, config.CookieDuration)
	for _, op := range userHandler.supportedOps {
		muxRouter.Handle(op.path, userHandler).Methods(op.method)
	}

	// Admin operations
	adminHandler := NewAdminHandler(db)
	for _, op := range adminHandler.supportedOps {
		muxRouter.Handle(op.path, adminHandler).Methods(op.method)
	}

	// Swagger operations
	swaggerHandler := NewSwaggerHandler()
	for _, op := range swaggerHandler.supportedOps {
		muxRouter.Handle(op.path, swaggerHandler).Methods(op.method)
	}

	fmt.Println("GoBoard version 0.0.1 starting on port", config.ListenPort)

	var handler http.Handler
	if fiAccessLog != nil {
		handler = handlers.LoggingHandler(fiAccessLog, muxRouter)
	} else {
		handler = muxRouter
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
