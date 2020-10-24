package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/gorilla/sessions"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var c Config // global var to hold static configuration

type File struct {
	Creator     string
	Name        string
	UpdatedTime time.Time
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

/// Perform some checks to make sure the file is OK
func checkIfValidFile(filename string, fileBytes []byte) error {
	ext := strings.ToLower(path.Ext(filename))
	found := false
	for _, mimetype := range c.OkExtensions {
		if ext == mimetype {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Invalid file extension: %s", ext)
	}
	if len(fileBytes) > c.MaxFileSize {
		return fmt.Errorf("File too large. File was %s bytes, Max file size is %s", len(fileBytes), c.MaxFileSize)
	}
	return nil
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
			result = append(result, &File{
				Name:        info.Name(),
				Creator:     path.Base(creatorFolder),
				UpdatedTime: info.ModTime(),
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

func main() {
	configPath := flag.String("c", "flounder.toml", "path to config file")
	var err error
	c, err = getConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	SessionStore = sessions.NewCookieStore([]byte(c.CookieStoreKey))
	DB, err = sql.Open("sqlite3", c.DBFile)
	if err != nil {
		log.Fatal(err)
	}

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
