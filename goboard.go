package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v2"
)

const goBoardVer = 0.03

// Config holds the configuration of the process
type Config struct {
	ListenPort        string      `yaml:"ListenPort"`
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
}

var upgrader = websocket.Upgrader{} // use default options

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
				log.Printf("error: %v", err)
				log.Printf("Fallbacking to stdout")
				fiAccessLog = os.Stdout
			}
		}
	} else {
		fiAccessLog = os.Stdout
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

	// Backend operations
	backendHandler := NewBackendHandler(config.MaxHistorySize, config.BackendTimeZone)
	backendHandler.Db = db
	for _, op := range backendHandler.supportedOps {
		r.Handle(op.RestPath, backendHandler).Methods(op.Method)
	}

	// User operations
	userHandler := NewUserHandler(config.CookieDuration)
	userHandler.Db = db
	for _, op := range userHandler.supportedOps {
		r.Handle(op.RestPath, userHandler).Methods(op.Method)
	}

	// Admin operations
	adminHandler := NewAdminHandler(config.AdminToken)
	adminHandler.Db = db
	for _, op := range adminHandler.supportedOps {
		r.Handle(op.RestPath, adminHandler).Methods(op.Method)
	}

	templateHandler := NewTemplateHandler()

	// Swagger operations
	if len(config.SwaggerPath) > 0 {

		// Sanity checks before enabling swagger capability
		realPath := os.ExpandEnv(config.SwaggerPath)
		if fi, err := os.Stat(realPath); os.IsNotExist(err) || !fi.IsDir() {
			log.Println(realPath, "Not found or not a directory: Disabling swagger capabilities")
		} else if _, err := os.Stat(strings.Join([]string{realPath, "/index.html"}, "")); os.IsNotExist(err) {
			log.Println(strings.Join([]string{realPath, "/index.html"}, ""), "Not found: Disabling swagger capabilities")
		} else {
			templateHandler.SetSwaggerBaseDir(realPath)
			swaggerOp := templateHandler.GetSwaggerOp()
			r.HandleFunc(swaggerOp.RestPath, swaggerOp.handler).Methods(swaggerOp.Method)
			r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(http.Dir(realPath))))
			r.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))

		}
	}

	// Webui operations
	if len(config.WebuiPath) > 0 {

		// Sanity checks before enabling webui capability
		realPath := os.ExpandEnv(config.WebuiPath)
		if fi, err := os.Stat(realPath); os.IsNotExist(err) || !fi.IsDir() {
			log.Println(realPath, "Not found or not a directory: Disabling webui capabilities")
		} else if _, err := os.Stat(strings.Join([]string{realPath, "/index.html"}, "")); os.IsNotExist(err) {
			log.Println(strings.Join([]string{realPath, "/index.html"}, ""), "Not found: Disabling webui capabilities")
		} else {
			templateHandler.setWebUIBaseDir(realPath)
			r.PathPrefix("/webui/").Handler(http.StripPrefix("/webui/", http.FileServer(http.Dir(realPath))))
			r.Handle("/webui", http.RedirectHandler("/webui/", http.StatusMovedPermanently))
			r.Handle("/", http.RedirectHandler("/webui/", http.StatusMovedPermanently))
		}
	}

	fmt.Println("GoBoard version ", goBoardVer, " starting on port", config.ListenPort)

	var handler http.Handler
	if fiAccessLog != nil {
		handler = handlers.LoggingHandler(fiAccessLog, mainRouter)
	} else {
		handler = mainRouter
	}

	server := &http.Server{Addr: fmt.Sprint(":", config.ListenPort), Handler: handler}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Setting up signal capturing
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT)

	// Waiting for SIGNAL
	s := <-sigs

	log.Printf("Signal '%s' received, exiting with 5s graceful-timeout", s.String())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
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
