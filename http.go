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

// TODO somewhat better error handling
const InternalServerError = "500: Internal Server Error"

func renderError(w http.ResponseWriter, errorMsg string) { // TODO think about pointers
	data := struct{ ErrorMsg string }{errorMsg}
	err := t.ExecuteTemplate(w, "error.html", data)
	if err != nil { // shouldn't happen probably
		fmt.Fprintf(w, errorMsg)
	}
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	indexFiles, err := getIndexFiles()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		renderError(w, InternalServerError)
		return
	}
	allUsers, err := getUsers()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		renderError(w, InternalServerError)
		return
	}
	data := struct {
		Domain    string
		PageTitle string
		Files     []*File
		Users     []string
	}{h.Domain, h.SiteTitle, indexFiles, allUsers}
	err = t.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		renderError(w, InternalServerError)
		return
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
