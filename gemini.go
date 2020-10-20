package main

import (
	"fmt"
	"git.sr.ht/~adnano/gmi"
)

func gmiIndex(w *gmi.ResponseWriter, r *gmi.Request) {
	fmt.Fprintf(w, "index")
}

func gmiPage(w *gmi.ResponseWriter, r *gmi.Request) {
}

func runGeminiServer() {
	var server gmi.Server
	server.HandleFunc("flounder.online", gmiIndex)
	server.ListenAndServe()
}
