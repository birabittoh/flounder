package main

import (
	"crypto/tls"
	"strings"
	// "fmt"
	"git.sr.ht/~adnano/gmi"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"text/template"
)

func gmiIndex(w *gmi.ResponseWriter, r *gmi.Request) {
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
	fileName := path.Join(c.FilesDirectory, userName, filepath.Clean(r.URL.Path))
	data, err := ioutil.ReadFile(fileName)
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

	if err := server.CertificateStore.Load("./tmpcerts"); err != nil {
		log.Fatal(err)
	}
	// is this necc?
	server.GetCertificate = func(hostname string, store *gmi.CertificateStore) *tls.Certificate {
		cert, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile)
		if err != nil {
			log.Fatal("Invalid TLS cert")
		}
		return &cert
	}

	// replace with wildcard cert
	hostname := strings.SplitN(c.Host, ":", 1)[0]
	server.HandleFunc(hostname, gmiIndex)
	server.HandleFunc("*."+hostname, gmiPage)

	server.ListenAndServe()
}
