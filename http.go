package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

var t *template.Template

// TODO somewhat better error handling
const InternalServerErrorMsg = "500: Internal Server Error"

func renderError(w http.ResponseWriter, errorMsg string, statusCode int) { // TODO think about pointers
	data := struct{ ErrorMsg string }{errorMsg}
	err := t.ExecuteTemplate(w, "error.html", data)
	if err != nil { // shouldn't happen probably
		http.Error(w, errorMsg, statusCode)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	indexFiles, err := getIndexFiles()
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
	allUsers, err := getUsers()
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
	data := struct {
		Domain    string
		PageTitle string
		Files     []*File
		Users     []string
	}{c.RootDomain, c.SiteTitle, indexFiles, allUsers}
	err = t.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
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

// Server a user's file
func userFile(w http.ResponseWriter, r *http.Request) {
	userName := strings.Split(r.Host, ".")[0]
	fileName := path.Join(c.FilesDirectory, userName, r.URL.Path)
	http.ServeFile(w, r, fileName)
}

func runHTTPServer() {
	log.Println("Running http server")
	var err error
	t, err = template.ParseGlob("./templates/*.html") // TODO make template dir configruable
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc(c.RootDomain+"/", indexHandler)
	http.HandleFunc(c.RootDomain+"/my_site", mySiteHandler)
	http.HandleFunc(c.RootDomain+"/edit/", editFileHandler)
	// http.HandleFunc("/delete/", deleteFileHandler)
	// login+register functions
	http.HandleFunc("/", userFile)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
