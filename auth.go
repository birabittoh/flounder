package main

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
	"strings"
)

func addUser(username string, password string) error {
	file, err := os.OpenFile(c.PasswdFile, os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), -1)
	if err != nil {
		return err
	}
	newUser := fmt.Sprintf("%s:%s\n", username, hash)
	file.WriteString(newUser)
	return nil
}
func checkAuth(username string, password string) error {
	file, err := os.OpenFile(c.PasswdFile, os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			return fmt.Errorf("malformed line, no colon: %s", line)
		}
		if username == parts[0] {
			return bcrypt.CompareHashAndPassword([]byte(parts[1]), []byte(password))
		}
	}
	return fmt.Errorf("User not found")
}
