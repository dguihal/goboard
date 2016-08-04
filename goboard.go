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

type Config struct {
	ListenPort        string      `yaml:"ListenPort"`
	MaxHistorySize    int         `yaml:"MaxHistorySize"`
	CookieDuration    int         `yaml:"CookieDuration"`
	GoBoardDBFile     string      `yaml:"GoBoardDBFile"`
	GoBoardDBFileMode os.FileMode `yaml:"GoBoardDBFileMode"`
	AccessLogFile     string      `yaml:"AccessLogFile"`
	AccessLogFileMode os.FileMode `yaml:"AccessLogFileMode"`
	SwaggerPath       string      `yaml:"SwaggerPath"`
	AdminToken        string      `yaml:"AdminToken"`
}

type RestEndpointHandler func(http.ResponseWriter, *http.Request)

type SupportedOp struct {
	PathBase string
	RestPath string
	Method   string
	handler  RestEndpointHandler
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
	if config.AccessLogFileMode == 0 {
		config.AccessLogFileMode = 0660
	}
	var fiAccessLog *os.File = nil
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

	muxRouter := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", Index)c

	// Backend operations
	backendHandler := NewBackendHandler(db, config.MaxHistorySize)
	for _, op := range backendHandler.supportedOps {
		muxRouter.Handle(op.RestPath, backendHandler).Methods(op.Method)
	}

	// User operations
	userHandler := NewUserHandler(db, config.CookieDuration)
	for _, op := range userHandler.supportedOps {
		muxRouter.Handle(op.RestPath, userHandler).Methods(op.Method)
	}

	// Admin operations
	adminHandler := NewAdminHandler(db, config.AdminToken)
	for _, op := range adminHandler.supportedOps {
		muxRouter.Handle(op.RestPath, adminHandler).Methods(op.Method)
	}

	// Swagger operations
	if len(config.SwaggerPath) > 0 {
		realPath := os.ExpandEnv(config.SwaggerPath)
		if _, err := os.Stat(strings.Join([]string{realPath, "/index.html"}, "")); os.IsNotExist(err) {
			log.Println(strings.Join([]string{realPath, "/index.html"}, ""), "Not found: Disabling swagger capabilities")
		} else {
			swaggerHandler := NewSwaggerHandler(realPath)
			for _, op := range swaggerHandler.supportedOps {
				muxRouter.Handle(op.RestPath, swaggerHandler).Methods(op.Method)
			}
		}
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
