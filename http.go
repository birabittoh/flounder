package main

import (
	"bytes"
	"database/sql"
	"fmt"
	gmi "git.sr.ht/~adnano/go-gemini"
	"github.com/gorilla/handlers"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var t *template.Template
var DB *sql.DB
var SessionStore *sessions.CookieStore

const InternalServerErrorMsg = "500: Internal Server Error"

func renderError(w http.ResponseWriter, errorMsg string, statusCode int) {
	data := struct {
		PageTitle string
		ErrorMsg  string
	}{"Error!", errorMsg}
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
	authd, _, isAdmin := getAuthUser(r)
	indexFiles, err := getIndexFiles()
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
	allUsers, err := getActiveUserNames()
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
	data := struct {
		Host      string
		PageTitle string
		Files     []*File
		Users     []string
		LoggedIn  bool
		IsAdmin   bool
	}{c.Host, c.SiteTitle, indexFiles, allUsers, authd, isAdmin}
	err = t.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
}

func editFileHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := SessionStore.Get(r, "cookie-session")
	authUser, ok := session.Values["auth_user"].(string)
	if !ok {
		renderError(w, "403: Forbidden", 403)
		return
	}
	fileName := filepath.Clean(r.URL.Path[len("/edit/"):])
	isText := strings.HasPrefix(mime.TypeByExtension(path.Ext(fileName)), "text")
	if !isText {
		renderError(w, "Not a text file, cannot be edited here", 400) // correct status code?
		return
	}
	filePath := path.Join(c.FilesDirectory, authUser, fileName)

	if r.Method == "GET" {
		err := checkIfValidFile(filePath, nil)
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), 400)
			return
		}
		// create directories if dne
		f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
		var fileBytes []byte
		if os.IsNotExist(err) {
			fileBytes = []byte{}
			err = nil
		} else {
			defer f.Close()
			fileBytes, err = ioutil.ReadAll(f)
		}
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
		fileBytes := []byte(r.Form.Get("file_text"))
		err := checkIfValidFile(filePath, fileBytes)
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), 400)
			return
		}
		// create directories if dne
		os.MkdirAll(path.Dir(filePath), os.ModePerm)
		err = ioutil.WriteFile(filePath, fileBytes, 0644)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
		newName := filepath.Clean(r.Form.Get("rename"))
		if newName != fileName {
			newPath := path.Join(c.FilesDirectory, authUser, newName)
			os.Rename(filePath, newPath)
		}
		http.Redirect(w, r, "/my_site", 303)
	}
}

func uploadFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		session, _ := SessionStore.Get(r, "cookie-session")
		authUser, ok := session.Values["auth_user"].(string)
		if !ok {
			renderError(w, "403: Forbidden", 403)
			return
		}
		r.ParseMultipartForm(10 << 6) // why does this not work
		file, fileHeader, err := r.FormFile("file")
		fileName := filepath.Clean(fileHeader.Filename)
		defer file.Close()
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), 400)
			return
		}
		dest, _ := ioutil.ReadAll(file)
		err = checkIfValidFile(fileName, dest)
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), 400)
			return
		}
		destPath := path.Join(c.FilesDirectory, authUser, fileName)

		f, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
		defer f.Close()
		io.Copy(f, bytes.NewReader(dest))
	}
	http.Redirect(w, r, "/my_site", 303)
}

// bool whether auth'd, string is auth user
func getAuthUser(r *http.Request) (bool, string, bool) {
	session, _ := SessionStore.Get(r, "cookie-session")
	user, ok := session.Values["auth_user"].(string)
	isAdmin, _ := session.Values["admin"].(bool)
	return ok, user, isAdmin
}
func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	authd, authUser, _ := getAuthUser(r)
	if !authd {
		renderError(w, "403: Forbidden", 403)
		return
	}
	fileName := filepath.Clean(r.URL.Path[len("/delete/"):])
	filePath := path.Join(c.FilesDirectory, authUser, fileName)
	if r.Method == "POST" {
		os.Remove(filePath) // suppress error
	}
	http.Redirect(w, r, "/my_site", 303)
}

func mySiteHandler(w http.ResponseWriter, r *http.Request) {
	authd, authUser, isAdmin := getAuthUser(r)
	if !authd {
		renderError(w, "403: Forbidden", 403)
		return
	}
	// check auth
	userFolder := path.Join(c.FilesDirectory, authUser)
	files, _ := getMyFilesRecursive(userFolder, authUser)
	data := struct {
		Host      string
		PageTitle string
		AuthUser  string
		Files     []*File
		LoggedIn  bool
		IsAdmin   bool
	}{c.Host, c.SiteTitle, authUser, files, authd, isAdmin}
	_ = t.ExecuteTemplate(w, "my_site.html", data)
}

func archiveHandler(w http.ResponseWriter, r *http.Request) {
	authd, authUser, _ := getAuthUser(r)
	if !authd {
		renderError(w, "403: Forbidden", 403)
		return
	}
	if r.Method == "GET" {
		userFolder := filepath.Join(c.FilesDirectory, filepath.Clean(authUser))
		err := zipit(userFolder, w)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}

	}
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
		row := DB.QueryRow("SELECT username, password_hash, active, admin FROM user where username = $1 OR email = $1", name)
		var db_password []byte
		var username string
		var active bool
		var isAdmin bool
		_ = row.Scan(&username, &db_password, &active, &isAdmin)
		if db_password != nil && !active {
			data := struct {
				Error     string
				PageTitle string
			}{"Your account is not active yet. Pending admin approval", c.SiteTitle}
			t.ExecuteTemplate(w, "login.html", data)
			return
		}
		if bcrypt.CompareHashAndPassword(db_password, []byte(password)) == nil {
			log.Println("logged in")
			session, _ := SessionStore.Get(r, "cookie-session")
			session.Values["auth_user"] = username
			session.Values["admin"] = isAdmin
			session.Save(r, w)
			http.Redirect(w, r, "/my_site", 303)
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
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := SessionStore.Get(r, "cookie-session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", 303)
}

const ok = "-0123456789abcdefghijklmnopqrstuvwxyz"

func isOkUsername(s string) bool {
	if len(s) < 1 {
		return false
	}
	if len(s) > 31 {
		return false
	}
	for _, char := range s {
		if !strings.Contains(ok, strings.ToLower(string(char))) {
			return false
		}
	}
	return true
}
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := struct {
			Host      string
			Errors    []string
			PageTitle string
		}{c.Host, nil, "Register"}
		err := t.ExecuteTemplate(w, "register.html", data)
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
	} else if r.Method == "POST" {
		r.ParseForm()
		email := r.Form.Get("email")
		password := r.Form.Get("password")
		errors := []string{}
		if !strings.Contains(email, "@") {
			errors = append(errors, "Invalid Email")
		}
		if r.Form.Get("password") != r.Form.Get("password2") {
			errors = append(errors, "Passwords don't match")
		}
		if len(password) < 6 {
			errors = append(errors, "Password is too short")
		}
		username := strings.ToLower(r.Form.Get("username"))
		if !isOkUsername(username) {
			errors = append(errors, "Username is invalid: can only contain letters, numbers and hypens. Maximum 32 characters.")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8) // TODO handle error
		if len(errors) == 0 {
			_, err = DB.Exec("insert into user (username, email, password_hash) values ($1, $2, $3)", username, email, string(hashedPassword))
			if err != nil {
				log.Println(err)
				errors = append(errors, "Username or email is already used")
			}
		}
		if len(errors) > 0 {
			data := struct {
				Host      string
				Errors    []string
				PageTitle string
			}{c.Host, errors, "Register"}
			t.ExecuteTemplate(w, "register.html", data)
		} else {
			data := struct {
				Host      string
				Message   string
				PageTitle string
			}{c.Host, "Registration complete! The server admin will approve your request before you can log in.", "Registration Complete"}
			t.ExecuteTemplate(w, "message.html", data)
		}
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	_, _, isAdmin := getAuthUser(r)
	if !isAdmin {
		renderError(w, "403: Forbidden", 403)
		return
	}
	allUsers, err := getUsers()
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
	data := struct {
		Users     []User
		LoggedIn  bool
		IsAdmin   bool
		PageTitle string
		Host      string
	}{allUsers, true, true, "Admin", c.Host}
	err = t.ExecuteTemplate(w, "admin.html", data)
	if err != nil {
		log.Println(err)
		renderError(w, InternalServerErrorMsg, 500)
		return
	}
}

func getFavicon(user string) string {
	faviconPath := path.Join(c.FilesDirectory, filepath.Clean(user), "favicon.txt")
	content, err := ioutil.ReadFile(faviconPath)
	if err != nil {
		return ""
	}
	strcontent := []rune(string(content))
	if len(strcontent) > 0 {
		return string(strcontent[0])
	}
	return ""
}

// Server a user's file
func userFile(w http.ResponseWriter, r *http.Request) {
	userName := filepath.Clean(strings.Split(r.Host, ".")[0]) // clean probably unnecessary
	p := filepath.Clean(r.URL.Path)
	if p == "/" {
		p = "index.gmi"
	}
	fileName := path.Join(c.FilesDirectory, userName, p)
	extension := path.Ext(fileName)
	if r.URL.Path == "/style.css" {
		http.ServeFile(w, r, path.Join(c.TemplatesDirectory, "static/style.css"))
	}
	query := r.URL.Query()
	_, raw := query["raw"]
	// dumb content negotiation
	acceptsGemini := strings.Contains(r.Header.Get("Accept"), "text/gemini")
	if !raw && !acceptsGemini && (extension == ".gmi" || extension == ".gemini") {
		_, err := os.Stat(fileName)
		if err != nil {
			renderError(w, "404: file not found", 404)
			return
		}
		file, _ := os.Open(fileName)

		htmlString := textToHTML(gmi.ParseText(file))
		favicon := getFavicon(userName)
		log.Println(favicon)
		data := struct {
			SiteBody  template.HTML
			Favicon   string
			PageTitle string
		}{template.HTML(htmlString), favicon, userName}
		t.ExecuteTemplate(w, "user_page.html", data)
	} else {
		http.ServeFile(w, r, fileName)
	}
}

func adminUserHandler(w http.ResponseWriter, r *http.Request) {
	_, _, isAdmin := getAuthUser(r)
	if r.Method == "POST" {
		if !isAdmin {
			renderError(w, "403: Forbidden", 403)
			return
		}
		components := strings.Split(r.URL.Path, "/")
		if len(components) < 5 {
			renderError(w, "Invalid action", 400)
			return
		}
		userName := components[3]
		action := components[4]
		var err error
		if action == "activate" {
			err = activateUser(userName)
		} else if action == "delete" {
			err = deleteUser(userName)
		}
		if err != nil {
			log.Println(err)
			renderError(w, InternalServerErrorMsg, 500)
			return
		}
		http.Redirect(w, r, "/admin", 303)
	}
}

func runHTTPServer() {
	log.Printf("Running http server with hostname %s on port %d. TLS enabled: %t", c.Host, c.HttpPort, c.HttpsEnabled)
	var err error
	t, err = template.ParseGlob(path.Join(c.TemplatesDirectory, "*.html"))
	if err != nil {
		log.Fatal(err)
	}
	serveMux := http.NewServeMux()

	s := strings.SplitN(c.Host, ":", 2)
	hostname := s[0]
	port := c.HttpPort

	serveMux.HandleFunc(hostname+"/", rootHandler)
	serveMux.HandleFunc(hostname+"/my_site", mySiteHandler)
	serveMux.HandleFunc(hostname+"/my_site/flounder-archive.zip", archiveHandler)
	serveMux.HandleFunc(hostname+"/admin", adminHandler)
	serveMux.HandleFunc(hostname+"/edit/", editFileHandler)
	serveMux.HandleFunc(hostname+"/upload", uploadFilesHandler)
	serveMux.HandleFunc(hostname+"/login", loginHandler)
	serveMux.HandleFunc(hostname+"/logout", logoutHandler)
	serveMux.HandleFunc(hostname+"/register", registerHandler)
	serveMux.HandleFunc(hostname+"/delete/", deleteFileHandler)

	// admin commands
	serveMux.HandleFunc(hostname+"/admin/user/", adminUserHandler)

	// TODO rate limit login https://github.com/ulule/limiter

	wrapped := handlers.LoggingHandler(log.Writer(), serveMux)

	// handle user files based on subdomain
	serveMux.HandleFunc("/", userFile)
	// login+register functions
	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         fmt.Sprintf(":%d", port),
		// TLSConfig:    tlsConfig,
		Handler: wrapped,
	}
	if c.HttpsEnabled {
		log.Fatal(srv.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile))
	} else {
		log.Fatal(srv.ListenAndServe())
	}
}
