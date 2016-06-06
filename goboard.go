// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	//html/template"
	//encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	//	"regexp"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ListenPort string
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

	restHandler, err := newRestHandler("my.db")
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter().StrictSlash(true)
	//router.HandleFunc("/", Index)
	router.Handle("/backend", restHandler).Methods("GET")
	router.Handle("/backend/{format}", restHandler).Methods("GET")
	router.Handle("/post", restHandler).Methods("POST")

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
