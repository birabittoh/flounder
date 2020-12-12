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
	user, pass, ok := r.BasicAuth()
	if ok && (checkAuth(user, pass) == nil) {
		webdavHandler := webdav.Handler{
			FileSystem: webdav.Dir(getUserDirectory(user)),
			Prefix:     "/webdav/",
			LockSystem: webdav.NewMemLS(),
			Logger: func(r *http.Request, e error) {
				fmt.Println(e)
			},
		}
		webdavHandler.ServeHTTP(w, r)
	} else {
		http.Error(w, "Authentication Error", http.StatusUnauthorized)
	}
}
