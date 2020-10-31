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
	"path/filepath"
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
		err := activateUser(username)
		log.Fatal(err)
	case "delete-user":
		username := os.Args[3]
		// TODO add confirmation
		err := deleteUser(username)
		log.Fatal(err)
	case "make-admin":
		username := os.Args[3]
		err := makeAdmin(username)
		log.Fatal(err)
	}
	// reset password

}

func makeAdmin(username string) error {
	_, err := DB.Exec("UPDATE user SET admin = true WHERE username = $1", username)
	if err != nil {
		return err
	}
	return nil
}

func activateUser(username string) error {
	_, err := DB.Exec("UPDATE user SET active = true WHERE username = $1", username)
	if err != nil {
		return err
	}
	log.Println("Activated user", username)
	baseIndex := `# Welcome to Flounder!
## About
Welcome to an ultra-lightweight platform for making and sharing small websites. You can get started by editing this page -- remove this content and replace it with whatever you like! It will be live at <your-name>.flounder.online. You can go there right now to see what this page currently looks like. Here is a link to a page which will give you more information about using flounder:
=> //admin.flounder.online

And here's a guide to the text format that Flounder uses to create pages, Gemini. These pages are converted into HTML so they can be displayed in a web browser.
=> //admin.flounder.online/gemini_text_guide.gmi

Have fun!`
	// Redundant filepath.Clean call just in case.
	username = filepath.Clean(username)
	os.Mkdir(path.Join(c.FilesDirectory, username), os.ModePerm)
	ioutil.WriteFile(path.Join(c.FilesDirectory, username, "index.gmi"), []byte(baseIndex), 0644)
	os.Mkdir(path.Join(c.FilesDirectory, username), os.ModePerm)
	return nil
}

func deleteUser(username string) error {
	_, err := DB.Exec("DELETE FROM user WHERE username = $1", username)
	if err != nil {
		return err
	}
	username = filepath.Clean(username)
	os.RemoveAll(path.Join(c.FilesDirectory, username))
	return nil
}
