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
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	router.Handle("/backend", backendHandler).Methods("GET") // Get backend (in xml)
	router.Handle("/backend/{format}", backendHandler).Methods("GET") //Get backend (in specific format)
	router.Handle("/post", backendHandler).Methods("POST") // Post new message

	// User operations
	userHandler := newUserHandler(db, config.CookieDuration)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	router.Handle("/user/add", userHandler).Methods("POST") // Add a user
	router.Handle("/user/login", userHandler).Methods("POST") // Sign in a user

	// Admin operations
	adminHandler := newAdminHandler(db)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	router.Handle("/admin/user", adminHandler).Methods("DELETE") // Delete a user
	router.Handle("/admin/post", adminHandler).Methods("POST") // Delete a post

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
//https://astaxie.gitbooks.io/build-web-application-with-golang/

//https://github.com/boltdb/bolt : Backend
//https://github.com/skyec/boltdb-server/blob/master/server.go
