package main

import (
	"fmt"
	"path"
	"strings"
	"time"
)

func timeago(t *time.Time) string {
	d := time.Since(*t)
	if d.Seconds() < 60 {
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	} else if d.Minutes() < 60 {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	} else if d.Hours() < 24 {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	} else {
		return fmt.Sprintf("%d days ago", int(d.Hours())/24)
	}
}

/// Perform some checks to make sure the file is OK
func checkIfValidFile(filename string, fileBytes []byte) error {
	ext := strings.ToLower(path.Ext(filename))
	found := false
	for _, mimetype := range c.OkExtensions {
		if ext == mimetype {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Invalid file extension: %s", ext)
	}
	if len(fileBytes) > c.MaxFileSize {
		return fmt.Errorf("File too large. File was %s bytes, Max file size is %s", len(fileBytes), c.MaxFileSize)
	}
	return nil
}
