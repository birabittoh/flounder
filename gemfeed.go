// Parses Gemfeed according to the companion spec: gemini://gemini.circumlunar.space/docs/companion/subscription.gmi
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Gemfeed struct {
	Title   string
	Creator string
	Url     *url.URL
	Entries []*FeedEntry
}

type FeedEntry struct {
	Title      string
	Url        *url.URL
	Date       time.Time
	DateString string
	Feed       *Gemfeed
	File       string // TODO refactor
}

// Non-standard extension
// Requires yyyy-mm-dd formatted files
func generateFeedFromUser(user string) []FeedEntry {
	gemlogFolder := "gemlog" // TODO make configurable
	gemlogFolderPath := path.Join(c.FilesDirectory, user, gemlogFolder)
	// NOTE: assumes sanitized input
	feed := Gemfeed{
		Title:   user + "'s Gemfeed",
		Creator: user,
		// URL? etc?
	}
	var feedEntries []FeedEntry
	err := filepath.Walk(gemlogFolderPath, func(thepath string, info os.FileInfo, err error) error {
		base := path.Base(thepath)
		if len(base) >= 10 {
			entry := FeedEntry{}
			date, err := time.Parse("2006-01-02", base[:10])
			if err != nil {
				return nil
			}
			entry.Date = date
			entry.DateString = base[:10]
			entry.Feed = &feed
			f, err := os.Open(thepath)
			if err != nil {
				return nil
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				// skip blank lines
				if scanner.Text() == "" {
					continue
				}
				line := scanner.Text()
				if strings.HasPrefix(line, "#") {
					entry.Title = strings.Trim(line, "# \t")
				} else {
					var title string
					if len(line) > 50 {
						title = line[:50]
					} else {
						title = line
					}
					entry.Title = "[" + title + "...]"
				}
				break
			}
			entry.File = getLocalPath(thepath)
			feedEntries = append(feedEntries, entry)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	sort.Slice(feedEntries, func(i, j int) bool {
		return feedEntries[i].Date.After(feedEntries[j].Date)
	})
	return feedEntries
}

// TODO definitely cache this function
// TODO include generateFeedFromFolder for "gemfeed" folders
func getAllGemfeedEntries() ([]*FeedEntry, []*Gemfeed, error) {
	maxUserItems := 25
	maxItems := 50
	var feedEntries []*FeedEntry
	var feeds []*Gemfeed
	users, err := getActiveUserNames()
	if err != nil {
		return nil, nil, err
	} else {
		for _, user := range users {
			fe := generateFeedFromUser(user)
			if len(fe) > 0 {
				feeds = append(feeds, fe[0].Feed)
				for _, e := range fe {
					feedEntries = append(feedEntries, &e)
				}
			}
		}
	}

	err = filepath.Walk(c.FilesDirectory, func(thepath string, info os.FileInfo, err error) error {
		if isGemini(info.Name()) {
			f, err := os.Open(thepath)
			// TODO verify no path bugs here
			creator := getCreator(thepath)
			baseUrl := url.URL{}
			baseUrl.Host = creator + "." + c.Host
			baseUrl.Path = getLocalPath(thepath)
			feed, err := ParseGemfeed(f, baseUrl, maxUserItems) // TODO make configurable
			f.Close()
			if err == nil {
				feed.Creator = creator
				if feed.Title == "" {
					feed.Title = "(Untitled Feed)"
				}
				feed.Url = &baseUrl
				feedEntries = append(feedEntries, feed.Entries...)
				feeds = append(feeds, feed)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	} else {
		sort.Slice(feedEntries, func(i, j int) bool {
			return feedEntries[i].Date.After(feedEntries[j].Date)
		})
		if len(feedEntries) > maxItems {
			return feedEntries[:maxItems], feeds, nil
		}
		return feedEntries, feeds, nil
	}
}

var GemfeedRegex = regexp.MustCompile(`=>\s*(\S+)\s([0-9]{4}-[0-9]{2}-[0-9]{2})\s?-?\s?(.*)`)

// Parsed Gemfeed text Returns error if not a gemfeed
// Doesn't sort output
// Doesn't get posts dated in the future
// if limit > -1 -- limit how many we are getting
func ParseGemfeed(text io.Reader, baseUrl url.URL, limit int) (*Gemfeed, error) {
	scanner := bufio.NewScanner(text)
	gf := Gemfeed{}
	for scanner.Scan() {
		if limit > -1 && len(gf.Entries) >= limit {
			break
		}
		line := scanner.Text()
		if gf.Title == "" && strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "##") {
			gf.Title = strings.Trim(line[1:], " \t")
		} else if strings.HasPrefix(line, "=>") {
			matches := GemfeedRegex.FindStringSubmatch(line)
			if len(matches) == 4 {
				parsedUrl, err := url.Parse(matches[1])
				if err != nil {
					continue
				}
				date, err := time.Parse("2006-01-02", matches[2])
				if err != nil {
					continue
				}
				title := matches[3]
				if parsedUrl.Host == "" {
					// Is relative link
					parsedUrl.Host = baseUrl.Host
					parsedUrl.Path = path.Join(path.Dir(baseUrl.Path), parsedUrl.Path)
				}
				parsedUrl.Scheme = ""
				if time.Now().After(date) {
					fe := FeedEntry{title, parsedUrl, date, matches[2], &gf, ""}
					if fe.Title == "" {
						fe.Title = "(Untitled)"
					}
					gf.Entries = append(gf.Entries, &fe)
				}
			}
		}
	}
	if len(gf.Entries) == 0 {
		return nil, fmt.Errorf("No Gemfeed entries found")
	}
	return &gf, nil
}
