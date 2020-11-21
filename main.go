package main

import (
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"github.com/gorilla/sessions"
	"io"
	"io/ioutil"
	"log"
	mathrand "math/rand"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var c Config // global var to hold static configuration

const HIDDEN_FOLDER = ".hidden"

type File struct { // also folders
	Creator     string
	Name        string // includes folder
	UpdatedTime time.Time
	TimeAgo     string
	IsText      bool
	Children    []*File
	Host        string
}

type User struct {
	Username  string
	Email     string
	Active    bool
	Admin     bool
	CreatedAt int // timestamp
}

// returns in a random order
func getActiveUserNames() ([]string, error) {
	rows, err := DB.Query(`SELECT username from user WHERE active is true`)
	if err != nil {
		return nil, err
	}
	var users []string
	for rows.Next() {
		var user string
		err = rows.Scan(&user)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	dest := make([]string, len(users))
	perm := mathrand.Perm(len(users))
	for i, v := range perm {
		dest[v] = users[i]
	}
	return dest, nil
}

func getUsers() ([]User, error) {
	rows, err := DB.Query(`SELECT username, email, active, admin, created_at from user ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	var users []User
	for rows.Next() {
		var user User
		err = rows.Scan(&user.Username, &user.Email, &user.Active, &user.Admin, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// get the user-reltaive local path from the filespath
// NOTE -- dont use on unsafe input ( I think )
func getLocalPath(filesPath string) string {
	l := len(strings.Split(c.FilesDirectory, "/"))
	return strings.Join(strings.Split(filesPath, "/")[l+1:], "/")
}

func getCreator(filePath string) string {
	l := len(strings.Split(c.FilesDirectory, "/"))
	r := strings.Split(filePath, "/")[l]
	return r
}

func getIndexFiles() ([]*File, error) { // cache this function
	result := []*File{}
	err := filepath.Walk(c.FilesDirectory, func(thepath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Failure accessing a path %q: %v\n", thepath, err)
			return err // think about
		}
		if info.IsDir() && info.Name() == HIDDEN_FOLDER {
			return filepath.SkipDir
		}
		// make this do what it should
		if !info.IsDir() {
			creatorFolder := getCreator(thepath)
			updatedTime := info.ModTime()
			result = append(result, &File{
				Name:        getLocalPath(thepath),
				Creator:     path.Base(creatorFolder),
				UpdatedTime: updatedTime,
				TimeAgo:     timeago(&updatedTime),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedTime.After(result[j].UpdatedTime)
	})
	if len(result) > 50 {
		result = result[:50]
	}
	return result, nil
} // todo clean up paths

func getMyFilesRecursive(p string, creator string) ([]*File, error) {
	result := []*File{}
	files, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		isText := strings.HasPrefix(mime.TypeByExtension(path.Ext(file.Name())), "text")
		fullPath := path.Join(p, file.Name())
		localPath := getLocalPath(fullPath)
		f := &File{
			Name:        localPath,
			Creator:     creator,
			UpdatedTime: file.ModTime(),
			IsText:      isText,
			Host:        c.Host,
		}
		if file.IsDir() {
			f.Children, err = getMyFilesRecursive(path.Join(p, file.Name()), creator)
		}
		result = append(result, f)
	}
	return result, nil
}

func createTablesIfDNE() {
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS user (
  id INTEGER PRIMARY KEY NOT NULL,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  active boolean NOT NULL DEFAULT false,
  admin boolean NOT NULL DEFAULT false,
  created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE IF NOT EXISTS cookie_key (
  value TEXT NOT NULL
);`)
	if err != nil {
		log.Fatal(err)
	}
}

// Generate a cryptographically secure key for the cookie store
func generateCookieKeyIfDNE() []byte {
	rows, err := DB.Query("SELECT value FROM cookie_key LIMIT 1")
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
	}
	if rows.Next() {
		var cookie []byte
		err := rows.Scan(&cookie)
		if err != nil {
			log.Fatal(err)
		}
		return cookie
	} else {
		k := make([]byte, 32)
		_, err := io.ReadFull(rand.Reader, k)
		if err != nil {
			log.Fatal(err)
		}
		_, err = DB.Exec("insert into cookie_key values ($1)", k)
		if err != nil {
			log.Fatal(err)
		}
		return k
	}
}

func main() {
	configPath := flag.String("c", "flounder.toml", "path to config file") // doesnt work atm
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("expected 'admin' or 'serve' subcommand")
		os.Exit(1)
	}

	var err error
	c, err = getConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	logFile, err := os.OpenFile(c.LogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	if c.HttpsEnabled {
		_, err1 := os.Stat(c.TLSCertFile)
		_, err2 := os.Stat(c.TLSKeyFile)
		if os.IsNotExist(err1) || os.IsNotExist(err2) {
			log.Fatal("Keyfile or certfile does not exist.")
		}
	}

	// Generate session cookie key if does not exist
	DB, err = sql.Open("sqlite3", c.DBFile)
	if err != nil {
		log.Fatal(err)
	}

	createTablesIfDNE()
	cookie := generateCookieKeyIfDNE()
	SessionStore = sessions.NewCookieStore(cookie)

	switch args[0] {
	case "serve":
		wg := new(sync.WaitGroup)
		wg.Add(2)
		go func() {
			runHTTPServer()
			wg.Done()
		}()
		go func() {
			runGeminiServer()
			wg.Done()
		}()
		wg.Wait()
	case "admin":
		runAdminCommand()
	}
}
