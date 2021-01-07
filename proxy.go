// Copied from https://git.sr.ht/~sircmpwn/kineto/tree/master/item/main.go
package main

import (
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~adnano/go-gemini"
)

func proxyGemini(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("404 Not found"))
		return
	}
	path := strings.SplitN(r.URL.Path, "/", 3)
	req := gemini.Request{}
	var err error
	req.Host = path[1]
	if len(path) > 2 {
		req.URL, err = url.Parse(fmt.Sprintf("gemini://%s/%s", path[1], path[2]))
	} else {
		req.URL, err = url.Parse(fmt.Sprintf("gemini://%s", path[1]))
	}
	client := gemini.Client{
		Timeout:           60 * time.Second,
		InsecureSkipTrust: true,
	}
	fmt.Println(req)

	if h := (url.URL{Host: req.Host}); h.Port() == "" {
		req.Host += ":1965"
	}

	resp, err := client.Do(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Gateway error: %v", err)
		return
	}
	defer resp.Body.Close()

	switch resp.Status {
	case 10, 11:
		// TODO accept input
		w.WriteHeader(http.StatusInternalServerError)
		return
	case 20:
		break // OK
	case 30, 31:
		to, err := url.Parse(resp.Meta)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(fmt.Sprintf("Gateway error: bad redirect %v", err)))
		}
		next := req.URL.ResolveReference(to)
		if next.Scheme != "gemini" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("This page is redirecting you to %s", next.String())))
			return
		}
		next.Host = r.URL.Host
		next.Scheme = r.URL.Scheme
		w.Header().Add("Location", next.String())
		w.WriteHeader(http.StatusFound)
		w.Write([]byte("Redirecting to " + next.String()))
		return
	case 40, 41, 42, 43, 44:
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "The remote server returned %d: %s", resp.Status, resp.Meta)
		return
	case 50, 51:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "The remote server returned %d: %s", resp.Status, resp.Meta)
		return
	case 52, 53, 59:
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "The remote server returned %d: %s", resp.Status, resp.Meta)
		return
	default:
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintf(w, "Proxy does not understand Gemini response status %d", resp.Status)
		return
	}

	m, _, err := mime.ParseMediaType(resp.Meta)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("Gateway error: %d %s: %v",
			resp.Status, resp.Meta, err)))
		return
	}

	if m != "text/gemini" {
		w.Header().Add("Content-Type", resp.Meta)
		io.Copy(w, resp.Body)
		return
	}

	w.Header().Add("Content-Type", "text/html")

	htmlString := textToHTML(gemini.ParseText(resp.Body))
	data := struct {
		SiteBody  template.HTML
		Favicon   string
		PageTitle string
		URI       string
	}{template.HTML(htmlString), "", req.URL.String(), req.URL.String()}

	err = t.ExecuteTemplate(w, "user_page.html", data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
}
