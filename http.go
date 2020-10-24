package main

import (
	"git.sr.ht/~adnano/gmi"
	"github.com/gorilla/handlers"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
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

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// serve everything inside static directory
	if r.URL.Path != "/" {
		fileName := path.Join(c.TemplatesDirectory, "static", filepath.Clean(r.URL.Path))
		http.ServeFile(w, r, fileName)
		return
	}
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
	authUser := "alex"
	fileName := filepath.Clean(r.URL.Path[len("/edit/"):])
	filePath := path.Join(c.FilesDirectory, authUser, fileName)
	if r.Method == "GET" {
		f, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0644)
		defer f.Close()
		fileBytes, err := ioutil.ReadAll(f)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
		data := struct {
			FileName  string
			FileText  string
			PageTitle string
		}{fileName, string(fileBytes), c.SiteTitle}
		err = t.ExecuteTemplate(w, "edit_file.html", data)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
	} else if r.Method == "POST" {
		// get post body
		r.ParseForm()
		fileText := r.Form.Get("file_text")
		err := ioutil.WriteFile(filePath, []byte(fileText), 0644)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
		http.Redirect(w, r, "/my_site", 302)
	}
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	authUser := "alex"
	fileName := filepath.Clean(r.URL.Path[len("/delete/"):])
	filePath := path.Join(c.FilesDirectory, authUser, fileName)
	if r.Method == "POST" {
		os.Remove(filePath)
		http.Redirect(w, r, "/my_site", 302)
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
	fileName := path.Join(c.FilesDirectory, userName, filepath.Clean(r.URL.Path))
	extension := path.Ext(fileName)
	if r.URL.Path == "/static/style.css" {
		http.ServeFile(w, r, path.Join(c.TemplatesDirectory, "static/style.css"))
	}
	if extension == ".gmi" || extension == ".gemini" {
		// covert to html
		stat, _ := os.Stat(fileName)
		file, _ := os.Open(fileName)
		htmlString := gmi.Parse(file).HTML()
		reader := strings.NewReader(htmlString)
		w.Header().Set("Content-Type", "text/html")
		http.ServeContent(w, r, fileName, stat.ModTime(), reader)
	} else {
		http.ServeFile(w, r, fileName)
	}
}

func runHTTPServer() {
	log.Println("Running http server")
	var err error
	t, err = template.ParseGlob("./templates/*.html") // TODO make template dir configruable
	if err != nil {
		log.Fatal(err)
	}
	serveMux := http.NewServeMux()

	serveMux.HandleFunc(c.RootDomain+"/", rootHandler)
	serveMux.HandleFunc(c.RootDomain+"/my_site", mySiteHandler)
	serveMux.HandleFunc(c.RootDomain+"/edit/", editFileHandler)
	// serveMux.HandleFunc(c.RootDomain+"/upload/", uploadFilesHandler)
	serveMux.HandleFunc(c.RootDomain+"/login", loginHandler)
	serveMux.HandleFunc(c.RootDomain+"/register", registerHandler)
	serveMux.HandleFunc(c.RootDomain+"/delete/", deleteFileHandler)

	// TODO rate limit login https://github.com/ulule/limiter

	wrapped := handlers.LoggingHandler(os.Stdout, serveMux)

	// handle user files based on subdomain
	serveMux.HandleFunc("/", userFile)
	// login+register functions
	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         ":8080",
		// TLSConfig:    tlsConfig,
		Handler: wrapped,
	}
	log.Fatal(srv.ListenAndServe())
}
