package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

const ( // todo make configurable
	userFilesPath = "./files"
)

type File struct {
	Creator     string
	Name        string
	UpdatedTime string
}

func getIndexFiles() ([]*File, error) { // cache this function
	result := []*File{}
	err := filepath.Walk(userFilesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Failure accessing a path %q: %v\n", path, err)
			return err // think about
		}
		// make this do what it should
		result = append(result, &File{
			Name:        info.Name(),
			Creator:     "alex",
			UpdatedTime: "123123",
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	// sort
	// truncate
	return result, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	indexFiles, _ := getIndexFiles()
	for _, file := range indexFiles {
		fmt.Fprintf(w, "%s\n", file.Name)
	}
}

func mySiteHandler(w http.ResponseWriter, r *http.Request) {
	authUser := "alex"
	files, _ := ioutil.ReadDir(path.Join(userFilesPath, authUser))
	for _, file := range files {
		fmt.Fprintf(w, "%s\n", file.Name())
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/my_site", mySiteHandler)
	// go serve gemini
	// go serve http
	log.Fatal(http.ListenAndServe(":8080", nil))
}
