package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

var t *template.Template

type IndexHandler struct {
	Domain    string
	SiteTitle string
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	indexFiles, err := getIndexFiles()
	if err != nil {
		log.Fatal(err)
	}
	allUsers, err := getUsers()
	if err != nil {
		log.Fatal(err)
	}
	data := struct {
		Domain    string
		PageTitle string
		Files     []*File
		Users     []string
	}{h.Domain, h.SiteTitle, indexFiles, allUsers}
	err = t.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		log.Fatal(err)
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

func runHTTPServer(config *Config) {
	var err error
	t, err = template.ParseGlob("./templates/*.html") // TODO make template dir configruable
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", &IndexHandler{config.RootDomain, config.SiteTitle})
	http.HandleFunc("/my_site", mySiteHandler)
	http.HandleFunc("/edit/", editFileHandler)
	// http.HandleFunc("/delete/", deleteFileHandler)
	// login+register functions
	log.Fatal(http.ListenAndServe(":8080", nil))
}
