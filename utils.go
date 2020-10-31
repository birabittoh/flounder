package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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
	if len(filename) == 0 {
		return fmt.Errorf("Please enter a filename")
	}
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
	fmt.Println(len(fileBytes))
	if len(fileBytes) > c.MaxFileSize {
		return fmt.Errorf("File too large. File was %d bytes, Max file size is %d", len(fileBytes), c.MaxFileSize)
	}
	return nil
}

func zipit(source string, target io.Writer) error {
	archive := zip.NewWriter(target)

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	archive.Close()

	return err
}
