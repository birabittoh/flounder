package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509" // todo move into cert file
	"encoding/pem"
	"strings"
	// "fmt"
	"git.sr.ht/~adnano/gmi"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"text/template"
	"time"
)

func gmiIndex(w *gmi.ResponseWriter, r *gmi.Request) {
	t, err := template.ParseFiles("templates/index.gmi")
	if err != nil {
		log.Fatal(err)
	}
	files, _ := getIndexFiles()
	users, _ := getUsers()
	data := struct {
		Domain    string
		SiteTitle string
		Files     []*File
		Users     []string
	}{
		Domain:    c.RootDomain,
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
	server.GetCertificate = func(hostname string, store *gmi.CertificateStore) *tls.Certificate {
		cert, err := store.Lookup(hostname)
		if err != nil {
			switch err {
			case gmi.ErrCertificateExpired:
				// Generate a new certificate if the current one is expired.
				log.Print("Old certificate expired, creating new one")
				fallthrough
			case gmi.ErrCertificateUnknown:
				// Generate a certificate if one does not exist.
				cert, err := gmi.NewCertificate(hostname, time.Minute)
				if err != nil {
					// Failed to generate new certificate, abort
					return nil
				}
				// Store and return the new certificate
				err = writeCertificate("./tmpcerts/"+hostname, cert)
				if err != nil {
					return nil
				}
				store.Add(hostname, cert)
				return &cert
			}
		}
		return cert
	}

	// replace with wildcard cert
	server.HandleFunc(c.RootDomain, gmiIndex)
	server.HandleFunc("*."+c.RootDomain, gmiPage)

	server.ListenAndServe()
}

// TODO log request

// writeCertificate writes the provided certificate and private key
// to path.crt and path.key respectively.
func writeCertificate(path string, cert tls.Certificate) error {
	crt, err := marshalX509Certificate(cert.Leaf.Raw)
	if err != nil {
		return err
	}
	key, err := marshalPrivateKey(cert.PrivateKey)
	if err != nil {
		return err
	}

	// Write the certificate
	crtPath := path + ".crt"
	crtOut, err := os.OpenFile(crtPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := crtOut.Write(crt); err != nil {
		return err
	}

	// Write the private key
	keyPath := path + ".key"
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := keyOut.Write(key); err != nil {
		return err
	}
	return nil
}

// marshalX509Certificate returns a PEM-encoded version of the given raw certificate.
func marshalX509Certificate(cert []byte) ([]byte, error) {
	var b bytes.Buffer
	if err := pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// marshalPrivateKey returns PEM encoded versions of the given certificate and private key.
func marshalPrivateKey(priv interface{}) ([]byte, error) {
	var b bytes.Buffer
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	if err := pem.Encode(&b, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
