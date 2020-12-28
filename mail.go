package main

import (
	"log"
	"net/smtp"
)

func SendEmail(email string, subject string, body string) {
	auth := smtp.PlainAuth("", c.SMTPUsername, c.SMTPPassword, "smtp.migadu.com")
	msg := "From: " + c.SMTPUsername + "\n" +
		"To: " + email + "\n" +
		"Cc: " + "alex@alexwennerberg.com" + "\n" + // TODO remove hardcode
		"Subject:" + subject + "\n" +
		body
	err := smtp.SendMail(c.SMTPServer, auth, c.SMTPUsername, []string{email}, []byte(msg))
	if err != nil {
		// doesnt need to block anything i think
		log.Println(err)
	}
}
