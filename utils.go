package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

func getSchemedFlounderLinkLines(r io.Reader) []string {
	scanner := bufio.NewScanner(r)
	result := []string{}
	for scanner.Scan() {
		text := scanner.Text()
		// TODO use actual parser. this could be a little wonky
		if strings.HasPrefix(text, "=>") && strings.Contains(text, c.Host) && (strings.Contains(text, "gemini://") || strings.Contains(text, "https://")) {
			result = append(result, text)
		}
	}
	return result
}

// Check if it is a text file, first by checking mimetype, then by reading bytes
// Stolen from https://github.com/golang/tools/blob/master/godoc/util/util.go
func isTextFile(fullPath string) bool {
	isText := strings.HasPrefix(mime.TypeByExtension(path.Ext(fullPath)), "text")
	if isText {
		return true
	}
	const max = 1024 // at least utf8.UTFMax
	s := make([]byte, 1024)
	f, err := os.Open(fullPath)
	if os.IsNotExist(err) {
		return true // for the purposes of editing, we return true
	}
	n, err := f.Read(s)
	s = s[0:n]
	if err != nil {
		return false
	}
	f.Close()

	for i, c := range string(s) {
		if i+utf8.UTFMax > len(s) {
			// last char may be incomplete - ignore
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' {
			// decoding error or control character - not a text file
			return false
		}
	}
	return true
}

// get the user-reltaive local path from the filespath
// NOTE -- dont use on unsafe input ( I think )
func getLocalPath(filesPath string) string {
	l := len(strings.Split(c.FilesDirectory, "/"))
	return strings.Join(strings.Split(filesPath, "/")[l+1:], "/")
}

func getCreator(filePath string) string {
	l := len(strings.Split(c.FilesDirectory, "/"))
	r := strings.Split(filePath, "/")[l]
	return r
}

func isGemini(filename string) bool {
	extension := path.Ext(filename)
	return extension == ".gmi" || extension == ".gemini"
}

func timeago(t *time.Time) string {
	d := time.Since(*t)
	if d.Seconds() < 60 {
		seconds := int(d.Seconds())
		if seconds == 1 {
			return "1 second ago"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	} else if d.Minutes() < 60 {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if d.Hours() < 24 {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(d.Hours()) / 24
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// safe
func getUserDirectory(username string) string {
	// extra filepath.clean just to be safe
	userFolder := path.Join(c.FilesDirectory, filepath.Clean(username))
	return userFolder
}

// ugh idk
func safeGetFilePath(username string, filename string) string {
	return path.Join(getUserDirectory(username), filepath.Clean(filename))
}

// TODO move into checkIfValidFile. rename it
func userHasSpace(user string, newBytes int) bool {
	userPath := path.Join(c.FilesDirectory, user)
	size, err := dirSize(userPath)
	if err != nil || size+int64(newBytes) > c.MaxUserBytes {
		return false
	}
	return true
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

/// Perform some checks to make sure the file is OK
func checkIfValidFile(filename string, fileBytes []byte) error {
	if len(filename) == 0 {
		return fmt.Errorf("Please enter a filename")
	}
	if len(filename) > 256 { // arbitrarily chosen
		return fmt.Errorf("Filename is too long")
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
	if len(fileBytes) > c.MaxFileBytes {
		return fmt.Errorf("File too large. File was %d bytes, Max file size is %d", len(fileBytes), c.MaxFileBytes)
	}
	//
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
