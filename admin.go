package main

// Commands for administering your instance
// reset user password -> generate link
// delete user

// Run some scripts to setup your instance

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// TODO improve cli
func runAdminCommand() {
	if len(os.Args) < 4 {
		fmt.Println("expected subcommand with parameter")
		os.Exit(1)
	}
	switch os.Args[2] {
	case "activate-user":
		username := os.Args[3]
		activateUser(username)
		// reset password
		// delete user (with are you sure?)
	}
}

func activateUser(username string) {
	_, err := DB.Exec("UPDATE user SET active = true WHERE username = $1", username)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Activated user", username)
	baseIndex := `# Welcome to Flounder!
## About
Flounder is an ultra-lightweight platform for making and sharing small websites. You can get started by editing this page -- remove this content and replace it with whatever you like! It will be live at <your-name>.flounder.online. You can go there right now to see what this page currently looks like. Here is a link to a page which will give you more information about using flounder:
=> https://admin.flounder.online

And here's a guide to the text format that Flounder uses to create pages, Gemini. These pages are converted into HTML so they can be displayed in a web browser.
=> https://admin.flounder.online/gemini_text_guide.gmi

Have fun!`
	os.Mkdir(path.Join(c.FilesDirectory, username), os.ModePerm)
	ioutil.WriteFile(path.Join(c.FilesDirectory, username, "index.gmi"), []byte(baseIndex), 0644)
	os.Mkdir(path.Join(c.FilesDirectory, username), os.ModePerm)
}
