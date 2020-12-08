package main

import (
	"fmt"
	"golang.org/x/net/webdav"
	"net/http"
)

func webdavHandler(w http.ResponseWriter, r *http.Request) {
	// get user
	if r.Header.Get("Authorization") == "" {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"wevdav\"")
		http.Error(w, "Authentication Error", http.StatusUnauthorized)
		return
	}
	for key, element := range r.Header {
		fmt.Println(key, element)
	}
	fmt.Println(r.Body)
	user, pass, ok := r.BasicAuth()
	if ok && (checkAuth(user, pass) == nil) {
		fmt.Println(user, pass)
		webdavHandler := webdav.Handler{
			FileSystem: webdav.Dir(getUserDirectory(user)),
			Prefix:     "/webdav/",
			LockSystem: webdav.NewMemLS(),
		}
		webdavHandler.ServeHTTP(w, r)
	} else {
		http.Error(w, "Authentication Error", http.StatusUnauthorized)
	}
}
