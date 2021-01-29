// Parses Gemfeed according to the companion spec:
// gemini://gemini.circumlunar.space/docs/companion/subscription.gmi
package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/feeds"
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
	Entries []FeedEntry
}

func (gf *Gemfeed) toAtomFeed() string {
	feed := feeds.Feed{
		Title:  gf.Title,
		Author: &feeds.Author{Name: gf.Creator},
		Link:   &feeds.Link{Href: gf.Url.String()},
	}
	feed.Items = []*feeds.Item{}
	for _, fe := range gf.Entries {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:   fe.Title,
			Link:    &feeds.Link{Href: fe.Url.String()}, // Rel=alternate?
			Created: fe.Date,                            // Updated not created?
		})
	}
	res, _ := feed.ToAtom()
	return res
}

type FeedEntry struct {
	Title      string
	Url        *url.URL
	Date       time.Time
	DateString string
	Feed       *Gemfeed
	File       string // TODO refactor
}

func urlFromPath(fullPath string) url.URL {
	creator := getCreator(fullPath)
	baseUrl := url.URL{}
	baseUrl.Host = creator + "." + c.Host
	baseUrl.Path = getLocalPath(fullPath)
	return baseUrl
}

// Non-standard extension
// Requires yyyy-mm-dd formatted files
func generateFeedFromUser(user string) *Gemfeed {
	gemlogFolderPath := path.Join(c.FilesDirectory, user, GemlogFolder)
	// NOTE: assumes sanitized input
	u := urlFromPath(gemlogFolderPath)
	feed := Gemfeed{
		Title:   strings.Title(user) + "'s Gemlog",
		Creator: user,
		Url:     &u,
	}
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
			u := urlFromPath(thepath)
			entry.Url = &u
			feed.Entries = append(feed.Entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	// Reverse chronological sort
	sort.Slice(feed.Entries, func(i, j int) bool {
		return feed.Entries[i].Date.After(feed.Entries[j].Date)
	})
	return &feed
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
					gf.Entries = append(gf.Entries, fe)
				}
			}
		}
	}
	if len(gf.Entries) == 0 {
		return nil, fmt.Errorf("No Gemfeed entries found")
	}
	return &gf, nil
}
