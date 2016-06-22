package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

const goboard_cookie_name string = "goboard_id"

const postBucketName string = "Posts"
const usersBucketName string = "Users"
const usersCookieBucketName string = "UsersCookie"

type Config struct {
	ListenPort     string `yaml:"ListenPort"`
	MaxHistorySize int    `yaml:"MaxHistorySize"`
	CookieDuration int    `yaml:"CookieDuration"`
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

	db, err := bolt.Open("my.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	router := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", Index)c

	// Backend operations
	backendHandler := newBackendHandler(db, config.MaxHistorySize)
	for _, op := range backendHandler.supportedOps {
		router.Handle(op.path, backendHandler).Methods(op.method)
	}

	// User operations
	userHandler := newUserHandler(db, config.CookieDuration)
	for _, op := range userHandler.supportedOps {
		router.Handle(op.path, userHandler).Methods(op.method)
	}

	// Admin operations
	adminHandler := newAdminHandler(db)
	for _, op := range adminHandler.supportedOps {
		router.Handle(op.path, adminHandler).Methods(op.method)
	}

	fmt.Println("GoBoard version 0.0.1 starting on port", config.ListenPort)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", config.ListenPort), router))
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
