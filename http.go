package main

import (
	"bytes"
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
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var t *template.Template
var SessionStore *sessions.CookieStore

func renderDefaultError(w http.ResponseWriter, statusCode int) {
	errorMsg := http.StatusText(statusCode)
	renderError(w, errorMsg, statusCode)
}

func renderError(w http.ResponseWriter, errorMsg string, statusCode int) {
	data := struct {
		StatusCode int
		ErrorMsg   string
		Config     Config
	}{statusCode, errorMsg, c}
	err := t.ExecuteTemplate(w, "error.html", data)
	if err != nil { // Shouldn't happen probably
		http.Error(w, errorMsg, statusCode)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// serve everything inside static directory
	if r.URL.Path != "/" {
		fileName := path.Join(c.TemplatesDirectory, "static", filepath.Clean(r.URL.Path))
		_, err := os.Stat(fileName)
		if err != nil {
			renderDefaultError(w, http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, fileName) // TODO better error handling
		return
	}

	user := newGetAuthUser(r)
	indexFiles, err := getIndexFiles(user.IsAdmin)
	if err != nil {
		panic(err)
	}
	allUsers, err := getActiveUserNames()
	if err != nil {
		panic(err)
	}
	data := struct {
		Config   Config
		AuthUser AuthUser
		Files    []*File
		Users    []string
	}{c, user, indexFiles, allUsers}
	err = t.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		panic(err)
	}
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	feedEntries, feeds, err := getAllGemfeedEntries()
	if err != nil {
		panic(err)
	}
	data := struct {
		Config      Config
		FeedEntries []FeedEntry
		Feeds       []Gemfeed
		AuthUser    AuthUser
	}{c, feedEntries, feeds, user}
	err = t.ExecuteTemplate(w, "feed.html", data)
	if err != nil {
		panic(err)
	}
}

func editFileHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	if !user.LoggedIn {
		renderDefaultError(w, http.StatusForbidden)
		return
	}
	fileName := filepath.Clean(r.URL.Path[len("/edit/"):])
	filePath := path.Join(c.FilesDirectory, user.Username, fileName)
	isText := isTextFile(filePath)
	alert := ""
	var warnings []string
	if r.Method == "POST" {
		// get post body
		r.ParseForm()
		fileText := r.Form.Get("file_text")
		// Web form by default gives us CR LF newlines.
		// Unix files use just LF
		fileText = strings.ReplaceAll(fileText, "\r\n", "\n")
		fileBytes := []byte(fileText)
		err := checkIfValidFile(filePath, fileBytes)
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), http.StatusBadRequest)
			return
		}
		sfl := getSchemedFlounderLinkLines(strings.NewReader(fileText))
		if len(sfl) > 0 {
			warnings = append(warnings, "Warning! Some of your links to flounder pages use schemas. This means that they may break when viewed in Gemini or over HTTPS. Plase remove gemini: or https: from the start of these links:\n")
			for _, l := range sfl {
				warnings = append(warnings, l)
			}
		}
		// create directories if dne
		os.MkdirAll(path.Dir(filePath), os.ModePerm)
		if userHasSpace(user.Username, len(fileBytes)) {
			if isText { // Cant edit binary files here
				err = ioutil.WriteFile(filePath, fileBytes, 0644)
			}
		} else {
			renderError(w, fmt.Sprintf("Bad Request: Out of file space. Max space: %d.", c.MaxUserBytes), http.StatusBadRequest)
			return
		}
		if err != nil {
			panic(err)
		}
		alert = "saved"
		newName := filepath.Clean(r.Form.Get("rename"))
		err = checkIfValidFile(newName, fileBytes)
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if newName != fileName {
			newPath := path.Join(c.FilesDirectory, user.Username, newName)
			os.MkdirAll(path.Dir(newPath), os.ModePerm)
			os.Rename(filePath, newPath)
			fileName = newName
			filePath = newPath
			alert += " and renamed"
		}
	}

	err := checkIfValidFile(filePath, nil)
	if err != nil {
		log.Println(err)
		renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Create directories if dne
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	var fileBytes []byte
	if os.IsNotExist(err) || !isText {
		fileBytes = []byte{}
		err = nil
	} else {
		defer f.Close()
		fileBytes, err = ioutil.ReadAll(f)
	}
	if err != nil {
		panic(err)
	}
	data := struct {
		FileName string
		FileText string
		Config   Config
		AuthUser AuthUser
		Host     string
		IsText   bool
		IsGemini bool
		IsGemlog bool
		Alert    string
		Warnings []string
	}{fileName, string(fileBytes), c, user, c.Host, isText, isGemini(fileName), strings.HasPrefix(fileName, "gemlog"), alert, warnings}
	err = t.ExecuteTemplate(w, "edit_file.html", data)
	if err != nil {
		panic(err)
	}
}

func uploadFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		user := newGetAuthUser(r)
		if !user.LoggedIn {
			renderDefaultError(w, http.StatusForbidden)
			return
		}
		r.ParseMultipartForm(10 << 6) // why does this not work
		file, fileHeader, err := r.FormFile("file")
		fileName := filepath.Clean(fileHeader.Filename)
		defer file.Close()
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), http.StatusBadRequest)
			return
		}
		dest, _ := ioutil.ReadAll(file)
		err = checkIfValidFile(fileName, dest)
		if err != nil {
			log.Println(err)
			renderError(w, err.Error(), http.StatusBadRequest)
			return
		}
		destPath := path.Join(c.FilesDirectory, user.Username, fileName)

		f, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if userHasSpace(user.Username, c.MaxFileBytes) { // Not quite right
			io.Copy(f, bytes.NewReader(dest))
		} else {
			renderError(w, fmt.Sprintf("Bad Request: Out of file space. Max space: %d.", c.MaxUserBytes), http.StatusBadRequest)
			return
		}
	}
	http.Redirect(w, r, "/my_site", http.StatusSeeOther)
}

type AuthUser struct {
	LoggedIn          bool
	Username          string
	IsAdmin           bool
	ImpersonatingUser string // used if impersonating
}

func newGetAuthUser(r *http.Request) AuthUser {
	session, _ := SessionStore.Get(r, "cookie-session")
	user, ok := session.Values["auth_user"].(string)
	impers, _ := session.Values["impersonating_user"].(string)
	isAdmin, _ := session.Values["admin"].(bool)
	return AuthUser{
		LoggedIn:          ok,
		Username:          user,
		IsAdmin:           isAdmin,
		ImpersonatingUser: impers,
	}
}

func mySiteHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	if !user.LoggedIn {
		renderDefaultError(w, http.StatusForbidden)
		return
	}
	// check auth
	userFolder := getUserDirectory(user.Username)
	files, _ := getMyFilesRecursive(userFolder, user.Username)
	currentDate := time.Now().Format("2006-01-02")
	data := struct {
		Config      Config
		Files       []File
		AuthUser    AuthUser
		CurrentDate string
	}{c, files, user, currentDate}
	_ = t.ExecuteTemplate(w, "my_site.html", data)
}

func myAccountHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	authUser := user.Username
	if !user.LoggedIn {
		renderDefaultError(w, http.StatusForbidden)
		return
	}
	me, _ := getUserByName(user.Username)
	type pageData struct {
		Config   Config
		AuthUser AuthUser
		MyUser   *User
		Errors   []string
	}
	data := pageData{c, user, me, nil}

	if r.Method == "GET" {
		err := t.ExecuteTemplate(w, "me.html", data)
		if err != nil {
			panic(err)
		}
	} else if r.Method == "POST" {
		r.ParseForm()
		newUsername := r.Form.Get("username")
		errors := []string{}
		newEmail := r.Form.Get("email")
		newDomain := r.Form.Get("domain")
		newUsername = strings.ToLower(newUsername)
		var err error
		_, exists := domains[newDomain]
		if newDomain != me.Domain && !exists {
			_, err = DB.Exec("update user set domain = ? where username = ?", newDomain, me.Username) // TODO use transaction
			if err != nil {
				errors = append(errors, err.Error())
			} else {
				refreshDomainMap()
				log.Printf("Changed domain for %s from %s to %s", authUser, me.Domain, newDomain)
			}
		}
		if newEmail != me.Email {
			_, err = DB.Exec("update user set email = ? where username = ?", newEmail, me.Username)
			if err != nil {
				// TODO better error not sql
				errors = append(errors, err.Error())
			} else {
				log.Printf("Changed email for %s from %s to %s", authUser, me.Email, newEmail)
			}
		}
		if newUsername != authUser {
			// Rename User
			err = renameUser(authUser, newUsername)
			if err != nil {
				log.Println(err)
				errors = append(errors, "Could not rename user")
			} else {
				session, _ := SessionStore.Get(r, "cookie-session")
				session.Values["auth_user"] = newUsername
				session.Save(r, w)
			}
		}
		// reset auth
		user = newGetAuthUser(r)
		data.Errors = errors
		data.AuthUser = user
		data.MyUser.Email = newEmail
		data.MyUser.Domain = newDomain
		_ = t.ExecuteTemplate(w, "me.html", data)
	}
}

func archiveHandler(w http.ResponseWriter, r *http.Request) {
	authUser := newGetAuthUser(r)
	if !authUser.LoggedIn {
		renderDefaultError(w, http.StatusForbidden)
		return
	}
	if r.Method == "GET" {
		userFolder := getUserDirectory(authUser.Username)
		err := zipit(userFolder, w)
		if err != nil {
			panic(err)
		}

	}
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// show page
		data := struct {
			Error  string
			Config Config
		}{"", c}
		err := t.ExecuteTemplate(w, "login.html", data)
		if err != nil {
			panic(err)
		}
	} else if r.Method == "POST" {
		r.ParseForm()
		name := strings.ToLower(r.Form.Get("username"))
		password := r.Form.Get("password")
		row := DB.QueryRow("SELECT username, password_hash, active, admin FROM user where username = $1 OR email = $1", name)
		var db_password []byte
		var username string
		var active bool
		var isAdmin bool
		err := row.Scan(&username, &db_password, &active, &isAdmin)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				data := struct {
					Error  string
					Config Config
				}{"Username or email '" + name + "' does not exist", c}
				t.ExecuteTemplate(w, "login.html", data)
				return
			} else {
				panic(err)
			}
		}
		if db_password != nil && !active {
			data := struct {
				Error  string
				Config Config
			}{"Your account is not active yet. Pending admin approval", c}
			t.ExecuteTemplate(w, "login.html", data)
			return
		}
		if bcrypt.CompareHashAndPassword(db_password, []byte(password)) == nil {
			log.Println("logged in")
			session, _ := SessionStore.Get(r, "cookie-session")
			session.Values["auth_user"] = username
			session.Values["admin"] = isAdmin
			session.Save(r, w)
			http.Redirect(w, r, "/my_site", http.StatusSeeOther)
		} else {
			data := struct {
				Error  string
				Config Config
			}{"Invalid login or password", c}
			err := t.ExecuteTemplate(w, "login.html", data)
			if err != nil {
				panic(err)
			}
		}
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := SessionStore.Get(r, "cookie-session")
	impers, ok := session.Values["impersonating_user"].(string)
	if ok {
		session.Values["auth_user"] = impers
		session.Values["impersonating_user"] = nil // TODO expire this automatically
		// session.Values["admin"] = nil // TODO fix admin
	} else {
		session.Options.MaxAge = -1
	}
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

const ok = "-0123456789abcdefghijklmnopqrstuvwxyz"

var bannedUsernames = []string{"www", "proxy", "grafana"}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := struct {
			Errors []string
			Config Config
		}{nil, c}
		err := t.ExecuteTemplate(w, "register.html", data)
		if err != nil {
			panic(err)
		}
	} else if r.Method == "POST" {
		r.ParseForm()
		email := strings.ToLower(r.Form.Get("email"))
		password := r.Form.Get("password")
		errors := []string{}
		if r.Form.Get("password") != r.Form.Get("password2") {
			errors = append(errors, "Passwords don't match")
		}
		if len(password) < 6 {
			errors = append(errors, "Password is too short")
		}
		username := strings.ToLower(r.Form.Get("username"))
		err := isOkUsername(username)
		if err != nil {
			errors = append(errors, err.Error())
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8) // TODO handle error
		if err != nil {
			panic(err)
		}
		reference := r.Form.Get("reference")
		if len(errors) == 0 {
			_, err = DB.Exec("insert into user (username, email, password_hash, reference) values ($1, $2, $3, $4)", username, email, string(hashedPassword), reference)
			if err != nil {
				errors = append(errors, "Username or email is already used")
			}
		}
		if len(errors) > 0 {
			data := struct {
				Config Config
				Errors []string
			}{c, errors}
			t.ExecuteTemplate(w, "register.html", data)
		} else {
			data := struct {
				Config  Config
				Message string
				Title   string
			}{c, "Registration complete! The server admin will approve your request before you can log in.", "Registration Complete"}
			t.ExecuteTemplate(w, "message.html", data)
		}
	}
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	if !user.LoggedIn {
		renderDefaultError(w, http.StatusForbidden)
		return
	}
	filePath := safeGetFilePath(user.Username, r.URL.Path[len("/delete/"):])
	if r.Method == "POST" {
		os.Remove(filePath) // TODO handle error
	}
	http.Redirect(w, r, "/my_site", http.StatusSeeOther)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	if !user.IsAdmin {
		renderDefaultError(w, http.StatusForbidden)
		return
	}
	allUsers, err := getUsers()
	if err != nil {
		log.Println(err)
		renderDefaultError(w, http.StatusInternalServerError)
		return
	}
	data := struct {
		Users    []User
		AuthUser AuthUser
		Config   Config
	}{allUsers, user, c}
	err = t.ExecuteTemplate(w, "admin.html", data)
	if err != nil {
		panic(err)
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
// TODO replace with gemini proxy
// Here be dragons
func userFile(w http.ResponseWriter, r *http.Request) {
	var userName string
	custom := domains[r.Host]
	if custom != "" {
		userName = custom
	} else {
		userName = filepath.Clean(strings.Split(r.Host, ".")[0]) // Clean probably unnecessary
	}
	p := filepath.Clean(r.URL.Path)
	var isDir bool
	fullPath := path.Join(c.FilesDirectory, userName, p) // TODO rename filepath
	stat, err := os.Stat(fullPath)
	if stat != nil {
		isDir = stat.IsDir()
	}
	if strings.HasSuffix(p, "index.gmi") {
		http.Redirect(w, r, path.Dir(p), http.StatusMovedPermanently)
	}

	if strings.HasPrefix(p, "/"+HiddenFolder) {
		renderDefaultError(w, http.StatusForbidden)
		return
	}
	if r.URL.Path == "/gemlog/atom.xml" && os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/atom+xml")
		// TODO set always somehow
		feed := generateFeedFromUser(userName)
		atomString := feed.toAtomFeed()
		io.Copy(w, strings.NewReader(atomString))
		return
	}

	var geminiContent string
	_, err = os.Stat(path.Join(fullPath, "index.gmi"))
	if isDir {
		// redirect slash
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, p+"/", http.StatusSeeOther)
		}
		if os.IsNotExist(err) {
			if p == "/gemlog" {
				geminiContent = generateGemfeedPage(userName)
			} else {
				geminiContent = generateFolderPage(fullPath)
			}
		} else {
			fullPath = path.Join(fullPath, "index.gmi")
		}
	}
	if geminiContent == "" && os.IsNotExist(err) {
		renderDefaultError(w, http.StatusNotFound)
		return
	}
	// Dumb content negotiation
	_, raw := r.URL.Query()["raw"]
	acceptsGemini := strings.Contains(r.Header.Get("Accept"), "text/gemini")
	if !raw && !acceptsGemini && (isGemini(fullPath) || geminiContent != "") {
		var htmlString string
		if geminiContent == "" {
			file, _ := os.Open(fullPath)
			htmlString = textToHTML(nil, gmi.ParseText(file))
			defer file.Close()
		} else {
			htmlString = textToHTML(nil, gmi.ParseText(strings.NewReader(geminiContent)))
		}
		favicon := getFavicon(userName)
		hostname := strings.Split(r.Host, ":")[0]
		uri := url.URL{
			Scheme: "gemini",
			Host:   hostname,
			Path:   p,
		}
		data := struct {
			SiteBody  template.HTML
			Favicon   string
			PageTitle string
			URI       *url.URL
		}{template.HTML(htmlString), favicon, userName + p, &uri}
		err = t.ExecuteTemplate(w, "user_page.html", data)
		if err != nil {
			panic(err)
		}
	} else {
		http.ServeFile(w, r, fullPath)
	}
}

func deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	if r.Method == "POST" {
		r.ParseForm()
		validate := r.Form.Get("validate-delete")
		if validate == user.Username {
			err := deleteUser(user.Username)
			if err != nil {
				log.Println(err)
				renderDefaultError(w, http.StatusInternalServerError)
				return
			}
			logoutHandler(w, r)
		} else {
			http.Redirect(w, r, "/me", http.StatusSeeOther)
		}
	}
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	data := struct {
		Config   Config
		AuthUser AuthUser
		Error    string
	}{c, user, ""}
	if r.Method == "GET" {
		err := t.ExecuteTemplate(w, "reset_pass.html", data)
		if err != nil {
			panic(err)
		}
	} else if r.Method == "POST" {
		r.ParseForm()
		enteredCurrPass := r.Form.Get("password")
		password1 := r.Form.Get("new_password1")
		password2 := r.Form.Get("new_password2")
		if password1 != password2 {
			data.Error = "New passwords do not match"
		} else if len(password1) < 6 {
			data.Error = "Password is too short"
		} else {
			err := checkAuth(user.Username, enteredCurrPass)
			if err == nil {
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password1), 8)
				if err != nil {
					panic(err)
				}
				_, err = DB.Exec("update user set password_hash = ? where username = ?", hashedPassword, user.Username)
				if err != nil {
					panic(err)
				}
				log.Printf("User %s reset password", user.Username)
				http.Redirect(w, r, "/me", http.StatusSeeOther)
				return
			} else {
				data.Error = "That's not your current password"
			}
		}
		err := t.ExecuteTemplate(w, "reset_pass.html", data)
		if err != nil {
			panic(err)
		}
	}
}

func adminUserHandler(w http.ResponseWriter, r *http.Request) {
	user := newGetAuthUser(r)
	if r.Method == "POST" {
		if !user.IsAdmin {
			renderDefaultError(w, http.StatusForbidden)
			return
		}
		components := strings.Split(r.URL.Path, "/")
		if len(components) < 5 {
			renderError(w, "Invalid action", http.StatusBadRequest)
			return
		}
		userName := components[3]
		action := components[4]
		var err error
		if action == "activate" {
			err = activateUser(userName)
		} else if action == "impersonate" {
			if user.ImpersonatingUser != "" {
				// Don't allow nested impersonation
				renderError(w, "Cannot nest impersonation, log out from impersonated user first.", 400)
				return
			}
			session, _ := SessionStore.Get(r, "cookie-session")
			session.Values["auth_user"] = userName
			session.Values["impersonating_user"] = user.Username
			session.Save(r, w)
			log.Printf("User %s impersonated %s", user.Username, userName)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Println(err)
			renderDefaultError(w, http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func checkDomainHandler(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain != "" && domains[domain] != "" {
		w.Write([]byte(domain))
		return
	}
	http.Error(w, "Not Found", 404)
}
func runHTTPServer() {
	log.Printf("Running http server with hostname %s on port %d.", c.Host, c.HttpPort)
	var err error
	t = template.New("main").Funcs(template.FuncMap{"parent": path.Dir, "hasSuffix": strings.HasSuffix})
	t, err = t.ParseGlob(path.Join(c.TemplatesDirectory, "*.html"))
	if err != nil {
		log.Fatal(err)
	}
	serveMux := http.NewServeMux()

	s := strings.SplitN(c.Host, ":", 2)
	hostname := s[0]
	port := c.HttpPort

	serveMux.HandleFunc(hostname+"/", rootHandler)
	serveMux.HandleFunc(hostname+"/feed", feedHandler)
	serveMux.HandleFunc(hostname+"/my_site", mySiteHandler)
	serveMux.HandleFunc(hostname+"/me", myAccountHandler)
	serveMux.HandleFunc(hostname+"/my_site/flounder-archive.zip", archiveHandler)
	serveMux.HandleFunc(hostname+"/admin", adminHandler)
	serveMux.HandleFunc(hostname+"/edit/", editFileHandler)
	serveMux.HandleFunc(hostname+"/upload", uploadFilesHandler)
	serveMux.Handle(hostname+"/login", limit(http.HandlerFunc(loginHandler)))
	serveMux.Handle(hostname+"/register", limit(http.HandlerFunc(registerHandler)))
	serveMux.HandleFunc(hostname+"/logout", logoutHandler)
	serveMux.HandleFunc(hostname+"/delete/", deleteFileHandler)
	serveMux.HandleFunc(hostname+"/delete-account", deleteAccountHandler)
	serveMux.HandleFunc(hostname+"/reset-password", resetPasswordHandler)

	// Check domain -- used by caddy
	serveMux.HandleFunc(hostname+"/check-domain", checkDomainHandler)

	// admin commands
	serveMux.HandleFunc(hostname+"/admin/user/", adminUserHandler)

	serveMux.HandleFunc(hostname+"/webdav/", webdavHandler)

	wrapped := handlers.CustomLoggingHandler(log.Writer(), handlers.RecoveryHandler()(serveMux), logFormatter)

	// handle user files based on subdomain
	// also routes to proxy
	serveMux.HandleFunc("proxy."+hostname+"/", proxyGemini) // eg. proxy.flounder.online
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
	log.Fatal(srv.ListenAndServe())
}
