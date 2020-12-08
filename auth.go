package main

import "golang.org/x/crypto/bcrypt"

func checkAuth(user string, password string) error {
	var actualPass []byte
	row := DB.QueryRow("SELECT password_hash FROM user where username = ?", user)
	err := row.Scan(&actualPass)
	if err != nil {
		return err
	}
	err = bcrypt.CompareHashAndPassword(actualPass, []byte(password))
	return err
}
