package main

import (
	"bufio"
	"database/sql"
	"fmt"
	gmi "git.sr.ht/~adnano/go-gemini"
	"github.com/gorilla/handlers"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Copy pasted from gorilla handler library, modified slightly

const lowerhex = "0123456789abcdef"

const apacheTS = "02/Jan/2006:15:04:05 -0700"

func logFormatter(writer io.Writer, params handlers.LogFormatterParams) {
	buf := buildCommonLogLine(params.Request, params.URL, params.TimeStamp, params.StatusCode, params.Size)
	buf = append(buf, '\n')
	writer.Write(buf)
}

// buildCommonLogLine builds a log entry for req in Apache Common Log Format.
// ts is the timestamp with which the entry should be logged.
// status and size are used to provide the response HTTP status and size.
func buildCommonLogLine(req *http.Request, url url.URL, ts time.Time, status int, size int) []byte {
	user := newGetAuthUser(req)
	username := "-"
	if user.Username != "" {
		username = user.Username
	}

	// Get forwarded IP address
	ipAddr := req.Header.Get("X-Real-IP")
	if ipAddr == "" {
		ipAddr = req.RemoteAddr
	}
	referer := req.Header.Get("Referer")

	host, _, err := net.SplitHostPort(ipAddr)
	if err != nil {
		host = ipAddr
	}

	uri := req.RequestURI

	// Requests using the CONNECT method over HTTP/2.0 must use
	// the authority field (aka r.Host) to identify the target.
	// Refer: https://httpwg.github.io/specs/rfc7540.html#CONNECT
	if req.ProtoMajor == 2 && req.Method == "CONNECT" {
		uri = req.Host
	}
	if uri == "" {
		uri = url.RequestURI()
	}

	desthost := req.Host

	buf := make([]byte, 0, 3*(len(host)+len(desthost)+len(username)+len(req.Method)+len(uri)+len(req.Proto)+len(referer)+50)/2)
	buf = append(buf, host...)
	buf = append(buf, " - "...)
	buf = append(buf, username...)
	buf = append(buf, " ["...)
	buf = append(buf, ts.Format(apacheTS)...)
	buf = append(buf, `] `...)
	buf = append(buf, desthost...)
	buf = append(buf, ` "`...)
	buf = append(buf, req.Method...)
	buf = append(buf, " "...)
	buf = appendQuoted(buf, uri)
	buf = append(buf, " "...)
	buf = append(buf, req.Proto...)
	buf = append(buf, `" - `...)
	buf = append(buf, referer...)
	buf = append(buf, " - "...)
	buf = append(buf, strconv.Itoa(status)...)
	buf = append(buf, " "...)
	buf = append(buf, strconv.Itoa(size)...)
	return buf
}

func appendQuoted(buf []byte, s string) []byte {
	var runeTmp [utf8.UTFMax]byte
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		if r == rune('"') || r == '\\' { // always backslashed
			buf = append(buf, '\\')
			buf = append(buf, byte(r))
			continue
		}
		if strconv.IsPrint(r) {
			n := utf8.EncodeRune(runeTmp[:], r)
			buf = append(buf, runeTmp[:n]...)
			continue
		}
		switch r {
		case '\a':
			buf = append(buf, `\a`...)
		case '\b':
			buf = append(buf, `\b`...)
		case '\f':
			buf = append(buf, `\f`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		case '\v':
			buf = append(buf, `\v`...)
		default:
			switch {
			case r < ' ':
				buf = append(buf, `\x`...)
				buf = append(buf, lowerhex[s[0]>>4])
				buf = append(buf, lowerhex[s[0]&0xF])
			case r > utf8.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				buf = append(buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			default:
				buf = append(buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	return buf
}

// Parse logs and write to database

// Anonymize user and IP?

func logGemini(r *gmi.Request) {
	ipAddr := r.RemoteAddr.String()
	host, _, err := net.SplitHostPort(ipAddr)
	if err != nil {
		host = ipAddr
	}
	line := fmt.Sprintf("gemini %s - [%s] %s %s\n", host,
		time.Now().Format(apacheTS),
		r.URL.Host,
		r.URL.Path)
	buf := []byte(line)
	log.Writer().Write(buf)
}

// notall fields set for both protocols
type LogLine struct {
	Timestamp time.Time
	Protocol  string // gemini or http
	ReqIP     string // maybe rename here
	ReqUser   string
	Status    int
	DestHost  string
	Method    string
	Referer   string
	Path      string
}

func (ll *LogLine) insertInto(db *sql.DB) {
	_, err := db.Exec(`insert into log (timestamp, protocol, request_ip, request_user, status, destination_host, path, method, referer)
values (?, ?, ?, ?, ?, ?, ?, ?, ?)`, ll.Timestamp.Format(time.RFC3339), ll.Protocol, ll.ReqIP, ll.ReqUser, ll.Status, ll.DestHost, ll.Path, ll.Method, ll.Referer)
	if err != nil {
		fmt.Println(err)
	}
}

const httpLogRegex = `^(.*?) - (.*?) \[(.*?)\] (.*?) \"(.*) (.*) .*\" - (.*) - (\d*)`
const geminiLogRegex = `^gemini (.*?) - \[(.*?)\] (.*?) (.*)`

var rxHttp *regexp.Regexp = regexp.MustCompile(httpLogRegex)
var rxGemini *regexp.Regexp = regexp.MustCompile(geminiLogRegex)

func lineToLogLine(line string) (*LogLine, error) {
	result := LogLine{}
	var ts string
	if strings.HasPrefix(line, "gemini") {
		matches := rxGemini.FindStringSubmatch(line)
		if len(matches) < 5 {
			return nil, nil // TODO better error
		} else {
			result.ReqIP = matches[1]
			ts = matches[2]
			result.Timestamp, _ = time.Parse(apacheTS, ts)
			result.DestHost = matches[3]
			result.Path = matches[4]
			result.Protocol = "gemini"
			// etc
		}
	} else {
		matches := rxHttp.FindStringSubmatch(line)
		if len(matches) < 8 {
			return nil, nil
		} else {
			result.ReqIP = matches[1]
			result.ReqUser = matches[2]
			ts = matches[3]
			result.Timestamp, _ = time.Parse(apacheTS, ts)
			result.DestHost = matches[4]
			result.Method = matches[5]
			result.Path = matches[6]
			result.Referer = matches[7]
			result.Status, _ = strconv.Atoi(matches[8])
			result.Protocol = "http"
		}
	}
	return &result, nil
}

func dumpLogs() {
	log.Println("Writing missing logs to database")
	db, err := getAnalyticsDB()
	if err != nil {
		// not perfect -- squashes errors
		return
	}
	var maxTime string
	row := db.QueryRow(`SELECT timestamp from log order by timestamp desc limit 1`)
	err = row.Scan(&maxTime)
	if err != nil {
		// not perfect -- squashes errors
		return
	}

	file, err := os.Open(c.LogFile)
	if err != nil {
		// not perfect -- squashes errors
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	counter := 0
	for scanner.Scan() {
		text := scanner.Text()
		ll, _ := lineToLogLine(text)
		if ll == nil {
			continue
		}
		if maxTime != "" {
			max, err := time.Parse(time.RFC3339, maxTime) // ineff
			if !ll.Timestamp.After(max) || err != nil {
				// NOTE -- possible bug if two requests in the same second while we are reading -- skips 1 log
				continue
			}
		}
		ll.insertInto(db)
		counter += 1
	}
	log.Printf("Wrote %d logs\n", counter)
}

func rotateLogs() {
	// TODO write
	// move log to log.1
	// delete log.1
}
