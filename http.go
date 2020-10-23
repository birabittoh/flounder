package main

import (
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
	// read file content. create if dne
	// authUser := "alex"
	data := struct {
		FileName  string
		FileText  string
		PageTitle string
	}{"filename", "filetext", c.SiteTitle}
	err := t.ExecuteTemplate(w, "edit_file.html", data)
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
}

func mySiteHandler(w http.ResponseWriter, r *http.Request) {
	authUser := "alex"
	// check auth
	files, _ := getUserFiles(authUser)
	data := struct {
		Domain    string
		PageTitle string
		AuthUser  string
		Files     []*File
	}{c.RootDomain, c.SiteTitle, authUser, files}
	_ = t.ExecuteTemplate(w, "my_site.html", data)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// show page
		data := struct {
			Error     string
			PageTitle string
		}{"", "Login"}
		err := t.ExecuteTemplate(w, "login.html", data)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
	} else if r.Method == "POST" {
		r.ParseForm()
		name := r.Form.Get("username")
		password := r.Form.Get("password")
		err := checkAuth(name, password)
		if err == nil {
			log.Println("logged in")
			// redirect home
		} else {
			data := struct {
				Error     string
				PageTitle string
			}{"Invalid login or password", c.SiteTitle}
			err := t.ExecuteTemplate(w, "login.html", data)
			if err != nil {
				log.Println(err)
				renderError(w, InternalServerErrorMsg, 500)
				return
			}
		}
		// create session
		// redirect home
		// verify login
		// check for errors
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := struct {
			Domain    string
			Errors    []string
			PageTitle string
		}{c.RootDomain, nil, "Register"}
		err := t.ExecuteTemplate(w, "register.html", data)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
	} else if r.Method == "POST" {
	}
}

// Server a user's file
func userFile(w http.ResponseWriter, r *http.Request) {
	userName := strings.Split(r.Host, ".")[0]
	fileName := path.Join(c.FilesDirectory, userName, r.URL.Path)
	// if gemini -- parse, convert, serve
	// else
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
	http.HandleFunc(c.RootDomain+"/login", loginHandler)
	http.HandleFunc(c.RootDomain+"/register", registerHandler)
	// http.HandleFunc("/delete/", deleteFileHandler)
	// login+register functions

	// handle user files based on subdomain
	http.HandleFunc("/", userFile)
	log.Fatal(http.ListenAndServe(":8080", logRequest(http.DefaultServeMux)))
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}
