package main

import (
	"crypto/rand"
	"database/sql"
	"flag"
	"github.com/gorilla/sessions"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var c Config // global var to hold static configuration

type File struct {
	Creator     string
	Name        string
	UpdatedTime time.Time
	TimeAgo     string
}

func getUsers() ([]string, error) {
	rows, err := DB.Query(`SELECT username from user`)
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

func getIndexFiles() ([]*File, error) { // cache this function
	result := []*File{}
	err := filepath.Walk(c.FilesDirectory, func(thepath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Failure accessing a path %q: %v\n", thepath, err)
			return err // think about
		}
		// make this do what it should
		if !info.IsDir() {
			creatorFolder, _ := path.Split(thepath)
			updatedTime := info.ModTime()
			result = append(result, &File{
				Name:        info.Name(),
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
	// sort
	// truncate
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedTime.Before(result[j].UpdatedTime)
	})
	if len(result) > 50 {
		result = result[:50]
	}
	return result, nil
} // todo clean up paths

func getUserFiles(user string) ([]*File, error) {
	result := []*File{}
	files, err := ioutil.ReadDir(path.Join(c.FilesDirectory, user))
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		result = append(result, &File{
			Name:        file.Name(),
			Creator:     user,
			UpdatedTime: file.ModTime(),
		})
	}
	return result, nil
}

func createTablesIfDNE() {
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS user (
  id INTEGER PRIMARY KEY NOT NULL,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  approved boolean NOT NULL DEFAULT false,
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
	configPath := flag.String("c", "flounder.toml", "path to config file")
	var err error
	c, err = getConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Generate self signed cert if does not exist. This is not suitable for production.
	_, err1 := os.Stat(c.TLSCertFile)
	_, err2 := os.Stat(c.TLSKeyFile)
	if os.IsNotExist(err1) || os.IsNotExist(err2) {
		log.Println("Keyfile or certfile does not exist.")
	}

	// Generate session cookie key if does not exist
	DB, err = sql.Open("sqlite3", c.DBFile)
	if err != nil {
		log.Fatal(err)
	}

	createTablesIfDNE()
	cookie := generateCookieKeyIfDNE()
	SessionStore = sessions.NewCookieStore(cookie)
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
}
