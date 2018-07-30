package main

import (
	"flag"
	"fmt"
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

const goBoardVer = 0.02

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

// Command line arguments management
var configFilePath string
var showHelp bool

func init() {
	const (
		defaultConfigFilePath = "goboard.yaml"
		usageC                = "Path to `goboard config file`"
		usageH                = "Show help"
	)
	flag.StringVar(&configFilePath, "config", defaultConfigFilePath, usageC)
	flag.StringVar(&configFilePath, "C", defaultConfigFilePath, usageC+" (shorthand)")
	flag.BoolVar(&showHelp, "help", false, usageH)
	flag.BoolVar(&showHelp, "h", false, usageH+" (shorthand)")
}

func main() {
	flag.Parse()

	println(configFilePath)
	println(showHelp)

	if showHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	log.Printf("Using %s as config file\n", configFilePath)

	configData, err := ioutil.ReadFile(configFilePath)
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

	// Set some failsafe defaults
	if config.GoBoardDBFileMode == 0 {
		config.GoBoardDBFileMode = 0600
	}
	if config.MaxHistorySize <= 0 {
		config.MaxHistorySize = 30
	}

	// Open database
	db, err := bolt.Open(config.GoBoardDBFile, config.GoBoardDBFileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Initialize router
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

	fmt.Println("GoBoard version ", goBoardVer, " starting on port", config.ListenPort)

	var handler http.Handler
	if fiAccessLog != nil {
		handler = handlers.LoggingHandler(fiAccessLog, mainRouter)
	} else {
		handler = mainRouter
	}
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", config.ListenPort), handler))
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
