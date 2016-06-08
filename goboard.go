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

type Config struct {
	ListenPort       string
	max_history_size int
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

	restHandler := newRestHandler(db, config.max_history_size)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	userHandler := newUserHandler(db)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	router := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", Index)
	router.Handle("/backend", restHandler).Methods("GET")
	router.Handle("/backend/{format}", restHandler).Methods("GET")
	router.Handle("/post", restHandler).Methods("POST")
	router.Handle("/user/add", userHandler).Methods("POST")
	router.Handle("/user/login", userHandler).Methods("POST")

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
