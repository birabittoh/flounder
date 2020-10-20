package main

import (
	"fmt"
	"log"
	"net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	indexFiles, _ := getIndexFiles()
	for _, file := range indexFiles {
		fmt.Fprintf(w, "%s\n", file.Name)
	}
}

func editFileHandler(w http.ResponseWriter, r *http.Request) {
	// get vs post
	// read file content
	authUser := "alex"
	files, _ := getUserFiles(authUser)
	for _, file := range files {
		fmt.Fprintf(w, "%s\n", file.Name)
	}
}

func mySiteHandler(w http.ResponseWriter, r *http.Request) {
	authUser := "alex"
	files, _ := getUserFiles(authUser)
	for _, file := range files {
		fmt.Fprintf(w, "%s\n", file.Name)
	}
}

func runHTTPServer() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/my_site", mySiteHandler)
	http.HandleFunc("/edit/", editFileHandler)
	// http.HandleFunc("/delete/", deleteFileHandler)
	// login+register functions
	log.Fatal(http.ListenAndServe(":8080", nil))
}
