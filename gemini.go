package main

import (
	"crypto/tls"
	"strings"
	// "fmt"
	gmi "git.sr.ht/~adnano/go-gemini"
	"log"
	"path"
	"path/filepath"
	"text/template"
	"time"
)

func gmiIndex(w *gmi.ResponseWriter, r *gmi.Request) {
	log.Println("Index request")
	t, err := template.ParseFiles("templates/index.gmi")
	if err != nil {
		log.Fatal(err)
	}
	files, err := getIndexFiles()
	users, err := getActiveUserNames()
	if err != nil {
		log.Println(err)
		w.WriteHeader(40, "Internal server error")
	}
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
	userName := filepath.Clean(strings.Split(r.URL.Host, ".")[0]) // clean probably unnecessary
	fileName := filepath.Clean(r.URL.Path)
	if fileName == "/" {
		fileName = "index.gmi"
	}
	log.Println("Request for gemini file", fileName, "for user", userName)

	gmi.ServeFile(w, gmi.Dir(path.Join(c.FilesDirectory, userName)), fileName)
}

func runGeminiServer() {
	log.Println("Starting gemini server")
	var server gmi.Server

	hostname := strings.SplitN(c.Host, ":", 2)[0]
	// is this necc?
	server.CreateCertificate = func(hostname string) (tls.Certificate, error) {
		log.Println("Generating certificate for", hostname)
		cert, err := gmi.CreateCertificate(gmi.CertificateOptions{
			DNSNames: []string{hostname},
			Duration: time.Minute, // for testing purposes
		})
		if err == nil {
			// Write the new certificate to disk
			err = writeCertificate(path.Join(c.GeminiCertStore, hostname), cert)
		}
		return cert, err
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
