package main

import (
	"crypto/tls"
	"strings"
	// "fmt"
	gmi "git.sr.ht/~adnano/go-gemini"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"text/template"
)

func gmiIndex(w *gmi.ResponseWriter, r *gmi.Request) {
	log.Println("Index request")
	t, err := template.ParseFiles("templates/index.gmi")
	if err != nil {
		log.Fatal(err)
	}
	files, _ := getIndexFiles()
	users, _ := getUsers()
	data := struct {
		Host      string
		SiteTitle string
		Files     []*File
		Users     []string
	}{
		Host:      c.Host,
		SiteTitle: c.SiteTitle,
		Files:     files,
		Users:     users,
	}
	t.Execute(w, data)
}

func gmiPage(w *gmi.ResponseWriter, r *gmi.Request) {
	userName := strings.Split(r.URL.Host, ".")[0]
	fileName := filepath.Clean(r.URL.Path)
	if fileName == "/" {
		fileName = "index.gmi"
	}
	filePath := path.Join(c.FilesDirectory, userName, fileName)
	log.Println("Request for gemini file at", filePath)
	data, err := ioutil.ReadFile(filePath)
	// serve file?
	// TODO write mimetype
	if err != nil {
		// TODO return 404 equivalent
		log.Fatal(err)
	}
	_, err = w.Write(data)
	if err != nil {
		// return internal server error
		log.Fatal(err)
	}
}

func runGeminiServer() {
	log.Println("Starting gemini server")
	var server gmi.Server

	hostname := strings.SplitN(c.Host, ":", 2)[0]
	// is this necc?
	server.GetCertificate = func(hostname string, store *gmi.CertificateStore) *tls.Certificate {
		cert, err := store.Lookup(hostname)
		if err != nil {
			cert, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile)
			if err != nil {
				log.Fatal("Invalid TLS cert")
			}
			store.Add(hostname, cert)
			return &cert
		}
		return cert
	}

	var mux gmi.ServeMux
	// replace with wildcard cert
	mux.HandleFunc("/", gmiIndex)

	var wildcardMux gmi.ServeMux
	wildcardMux.HandleFunc("/", gmiPage)
	server.Register(hostname, &mux)
	server.Register("*."+hostname, &wildcardMux)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
