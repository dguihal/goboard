package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	bolt "go.etcd.io/bbolt"
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

func loadConfig(path string) (*Config, error) {
	log.Printf("Using %s as config file\n", path)

	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", path, err)
	}

	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse config file %s: %w", path, err)
	}

	// Set some failsafe defaults
	if config.GoBoardDBFileMode == 0 {
		config.GoBoardDBFileMode = 0600
	}
	if config.MaxHistorySize <= 0 {
		config.MaxHistorySize = 30
	}
	if config.AccessLogFileMode == 0 {
		config.AccessLogFileMode = 0660
	}

	return &config, nil
}

func setupLogging(logFile string, mode os.FileMode) *os.File {
	var fiAccessLog *os.File
	var err error
	if len(logFile) > 0 {
		fiAccessLog, err = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND, mode)
		if err != nil {
			fiAccessLog, err = os.Create(logFile)
			if err != nil {
				log.Printf("error creating log file %s: %v", logFile, err)
				log.Printf("Fallbacking to stdout")
				return os.Stdout
			}
		}
	} else {
		fiAccessLog = os.Stdout
	}
	return fiAccessLog
}

func setupSwagger(r *mux.Router, templateHandler *TemplateHandler, swaggerPath string) {
	if len(swaggerPath) == 0 {
		return
	}
	// Sanity checks before enabling swagger capability
	realPath := os.ExpandEnv(swaggerPath)
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

func setupWebui(r *mux.Router, templateHandler *TemplateHandler, webuiPath string) {
	if len(webuiPath) == 0 {
		return
	}
	// Sanity checks before enabling webui capability
	realPath := os.ExpandEnv(webuiPath)
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

func setupRouter(db *bolt.DB, config *Config) *mux.Router {
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
	setupSwagger(r, templateHandler, config.SwaggerPath)
	setupWebui(r, templateHandler, config.WebuiPath)

	return mainRouter
}

func main() {
	flag.Parse()

	if showHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	config, err := loadConfig(configFilePath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Manage access log file
	fiAccessLog := setupLogging(config.AccessLogFile, config.AccessLogFileMode)
	if fiAccessLog != os.Stdout {
		defer fiAccessLog.Close()
	}

	// Open database
	db, err := bolt.Open(config.GoBoardDBFile, config.GoBoardDBFileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer db.Close()

	// Initialize router
	mainRouter := setupRouter(db, config)

	fmt.Println("GoBoard version ", goBoardVer, " starting on port", config.ListenPort)

	handler := handlers.LoggingHandler(fiAccessLog, mainRouter)

	server := &http.Server{
		Addr:              fmt.Sprint(":", config.ListenPort),
		Handler:           handler,
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       1 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       5 * time.Minute,
	}

	go func() {
		log.Println("Server starting on", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
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
		log.Printf("Server shutdown error: %v", err)
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
