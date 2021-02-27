package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var DB *sql.DB

func initializeDB() {
	var err error
	DB, err = sql.Open("sqlite3", c.DBFile)
	if err != nil {
		log.Fatal(err)
	}
	createTablesIfDNE()
}

// returns nil if login OK, err otherwise
// log in with email or username
func checkLogin(name string, password string) (string, bool, error) {
	row := DB.QueryRow("SELECT username, password_hash, active, admin FROM user where username = $1 OR email = $1", name)
	var db_password []byte
	var username string
	var active bool
	var isAdmin bool
	err := row.Scan(&username, &db_password, &active, &isAdmin)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return username, isAdmin, fmt.Errorf("Username or email '" + name + "' does not exist")
		} else {
			return username, isAdmin, err
		}
	}
	if db_password != nil && !active {
		return username, isAdmin, fmt.Errorf("Your account is not active yet. Pending admin approval %v", c)
	}
	if bcrypt.CompareHashAndPassword(db_password, []byte(password)) == nil {
		return username, isAdmin, nil
	} else {
		return username, isAdmin, fmt.Errorf("Invalid password")
	}
}

func getAnalyticsDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", c.AnalyticsDBFile)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS log (
  id INTEGER PRIMARY KEY NOT NULL,
  timestamp TEXT NOT NULL,
  protocol TEXT NOT NULL,
  request_ip TEXT,
  request_user TEXT,
  status INTEGER,
  destination_host TEXT,
  path TEXT,
  method TEXT,
  referer TEXT
);`)
	return db, err
}

type File struct { // also folders
	Creator     string
	Name        string // includes folder
	UpdatedTime time.Time
	TimeAgo     string
	IsText      bool
	Children    []File
	Host        string
}

func fileFromPath(fullPath string) File {
	info, _ := os.Stat(fullPath)
	creatorFolder := getCreator(fullPath)
	isText := isTextFile(fullPath)
	updatedTime := info.ModTime()
	return File{
		Name:        getLocalPath(fullPath),
		Creator:     path.Base(creatorFolder),
		UpdatedTime: updatedTime,
		IsText:      isText,
		TimeAgo:     timeago(&updatedTime),
		Host:        c.Host,
	}

}

type User struct {
	Username      string
	Email         string
	Active        bool
	Admin         bool
	CreatedAt     int64 // timestamp
	Reference     string
	Domain        string
	DomainEnabled bool
}

func getActiveUserNames() ([]string, error) {
	rows, err := DB.Query(`SELECT username from user WHERE active is true order by username`)
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

	return users, nil
}

var domains map[string]string

func refreshDomainMap() error {
	domains = make(map[string]string)
	rows, err := DB.Query(`SELECT domain, username from user WHERE domain != ""`)
	if err != nil {
		log.Println(err)
		return err
	}
	for rows.Next() {
		var domain string
		var username string
		err = rows.Scan(&domain, &username)
		if err != nil {
			return err
		}
		domains[domain] = username
	}
	return nil
}

func getUserByName(username string) (*User, error) {
	var user User
	row := DB.QueryRow(`SELECT username, email, active, admin, created_at, reference, domain, domain_enabled from user WHERE username = ?`, username)
	err := row.Scan(&user.Username, &user.Email, &user.Active, &user.Admin, &user.CreatedAt, &user.Reference, &user.Domain, &user.DomainEnabled)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func getUsers() ([]User, error) {
	rows, err := DB.Query(`SELECT username, email, active, admin, created_at, reference, domain from user ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	var users []User
	for rows.Next() {
		var user User
		err = rows.Scan(&user.Username, &user.Email, &user.Active, &user.Admin, &user.CreatedAt, &user.Reference, &user.Domain)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func getIndexFiles(admin bool) ([]*File, error) { // cache this function
	result := []*File{}
	err := filepath.Walk(c.FilesDirectory, func(thepath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Failure accessing a path %q: %v\n", thepath, err)
			return err // think about
		}
		if !admin && info.IsDir() && info.Name() == HiddenFolder {
			return filepath.SkipDir
		}
		// make this do what it should
		if !info.IsDir() {
			res := fileFromPath(thepath)
			result = append(result, &res)
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

func getMyFilesRecursive(p string, creator string) ([]File, error) {
	result := []File{}
	files, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fullPath := path.Join(p, file.Name())
		f := fileFromPath(fullPath)
		if file.IsDir() {
			f.Children, err = getMyFilesRecursive(path.Join(p, file.Name()), creator)
		}
		result = append(result, f)
	}
	return result, nil
}

func createTablesIfDNE() {
	_, err := DB.Exec(`CREATE TABLE user (
  id INTEGER PRIMARY KEY NOT NULL,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  reference TEXT NOT NULL default "",
  active boolean NOT NULL DEFAULT false,
  admin boolean NOT NULL DEFAULT false,
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  domain TEXT NOT NULL default "",
  domain_enabled BOOLEAN NOT NULL DEFAULT false
);`)
	if err == nil {
		// on first creation, create admin user with pw admin
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), 8) // TODO handle error
		if err != nil {
			log.Fatal(err)
		}
		_, err = DB.Exec(`INSERT OR IGNORE INTO user (username, email, password_hash, admin) values ('admin', 'default@flounder.local', ?, true)`, hashedPassword)
		activateUser("admin")
		if err != nil {
			log.Fatal(err)
		}
	}

	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS cookie_key (
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
		_, err = DB.Exec("insert into cookie_key values (?)", k)
		if err != nil {
			log.Fatal(err)
		}
		return k
	}
}
