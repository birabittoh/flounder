package main

// Commands for administering your instance
// reset user password -> generate link
// delete user

// Run some scripts to setup your instance

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

// TODO improve cli
func runAdminCommand() {
	args := flag.Args() // again?
	if len(args) < 3 {
		fmt.Println("Expected subcommand with parameter activate-user|delete-user|make-admin|rename-user")
		os.Exit(1)
	}
	var err error
	switch args[1] {
	case "activate-user":
		username := args[2]
		err = activateUser(username)
	case "delete-user":
		username := args[2]
		// TODO add confirmation
		err = deleteUser(username)
	case "make-admin":
		username := args[2]
		err = makeAdmin(username)
	case "rename-user":
		username := args[2]
		newUsername := args[3]
		err = renameUser(username, newUsername)
		// case "set-password":
	}
	if err != nil {
		log.Fatal(err)
	}
	// reset password

}

func makeAdmin(username string) error {
	_, err := DB.Exec("UPDATE user SET admin = true WHERE username = $1", username)
	if err != nil {
		return err
	}
	log.Println("Made admin user", username)
	return nil
}

func setPassword(username string, newPass string) error {
	return nil
}

func activateUser(username string) error {
	_, err := DB.Exec("UPDATE user SET active = true WHERE username = ?", username)
	if err != nil {
		// TODO verify 1 row updated
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

func renameUser(oldUsername string, newUsername string) error {
	err := isOkUsername(newUsername)
	if err != nil {
		return err
	}
	res, err := DB.Exec("UPDATE user set username = ? WHERE username = ?", newUsername, oldUsername)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if rowsAffected != 1 {
		return fmt.Errorf("No User updated %s %s", oldUsername, newUsername)
	} else if err != nil {
		return err
	}
	userFolder := path.Join(c.FilesDirectory, oldUsername)
	newUserFolder := path.Join(c.FilesDirectory, newUsername)
	err = os.Rename(userFolder, newUserFolder)
	if err != nil {
		// This would be bad. User in broken, insecure state.
		// TODO some sort of better handling?
		return err
	}
	log.Printf("Changed username from %s to %s", oldUsername, newUsername)
	return nil
}

func deleteUser(username string) error {
	_, err := DB.Exec("DELETE FROM user WHERE username = $1", username)
	if err != nil {
		return err
	}
	username = filepath.Clean(username)
	err = os.RemoveAll(path.Join(c.FilesDirectory, username))
	if err != nil {
		// bad state
		return err
	}
	log.Println("Deleted user", username)
	return nil
}
