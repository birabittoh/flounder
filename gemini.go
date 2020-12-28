package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509/pkix"
	gmi "git.sr.ht/~adnano/go-gemini"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var gt *template.Template

func generateGemfeedPage(user string) string {
	return ""
}

func generateFolderPage(fullpath string) string {
	files, _ := ioutil.ReadDir(fullpath)
	var renderedFiles = []File{}
	for _, file := range files {
		// Very awkward
		res := fileFromPath(path.Join(fullpath, file.Name()))
		renderedFiles = append(renderedFiles, res)
	}
	var buff bytes.Buffer
	data := struct {
		Host   string
		Folder string
		Files  []File
	}{c.Host, getLocalPath(fullpath), renderedFiles}
	err := gt.ExecuteTemplate(&buff, "folder.gmi", data)
	if err != nil {
		log.Println(err)
		return ""
	}
	return buff.String()
}

func gmiIndex(w *gmi.ResponseWriter, r *gmi.Request) {
	log.Println("Index request")
	t, err := template.ParseFiles("templates/index.gmi")
	if err != nil {
		log.Fatal(err)
	}
	files, err := getIndexFiles(false)
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
	} else if strings.HasPrefix(fileName, "/.hidden") {
		w.WriteStatus(gmi.StatusNotFound)
		return
	}
	log.Println("Request for gemini file", fileName, "for user", userName)

	gmi.ServeFile(w, gmi.Dir(path.Join(c.FilesDirectory, userName)), fileName)
}

func runGeminiServer() {
	log.Println("Starting gemini server")
	var err error
	gt, err = template.ParseGlob(path.Join(c.TemplatesDirectory, "*.gmi"))
	if err != nil {
		log.Fatal(err)
	}
	var server gmi.Server
	server.ReadTimeout = 1 * time.Minute
	server.WriteTimeout = 2 * time.Minute

	hostname := strings.SplitN(c.Host, ":", 2)[0]
	// is this necc?
	err = server.Certificates.Load(c.GeminiCertStore)
	if err != nil {
	}
	server.CreateCertificate = func(h string) (tls.Certificate, error) {
		log.Println("Generating certificate for", h)
		return gmi.CreateCertificate(gmi.CertificateOptions{
			Subject: pkix.Name{
				CommonName: hostname,
			},
			DNSNames: []string{h},
			Duration: time.Hour * 760, // one month
		})
	}

	var mux gmi.ServeMux
	// replace with wildcard cert
	mux.HandleFunc("/", gmiIndex)

	var wildcardMux gmi.ServeMux
	wildcardMux.HandleFunc("/", gmiPage)
	server.Register(hostname, &mux)
	server.Register("*."+hostname, &wildcardMux)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
