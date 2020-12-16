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
	"sort"
	"strings"
	"time"
)

type Gemfeed struct {
	Title   string
	Creator string
	FeedUrl *url.URL
	Entries []*FeedEntry
}

type FeedEntry struct {
	Title      string
	Url        *url.URL
	Date       time.Time
	DateString string
	Feed       *Gemfeed
}

// TODO definitely cache this function -- it reads EVERY gemini file on flounder.
func getAllGemfeedEntries() ([]*FeedEntry, error) {
	maxUserItems := 25
	maxItems := 100
	var feedEntries []*FeedEntry
	err := filepath.Walk(c.FilesDirectory, func(thepath string, info os.FileInfo, err error) error {
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
				feed.FeedUrl = &baseUrl
				feedEntries = append(feedEntries, feed.Entries...)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	} else {
		sort.Slice(feedEntries, func(i, j int) bool {
			return feedEntries[i].Date.After(feedEntries[j].Date)
		})
		if len(feedEntries) > maxItems {
			return feedEntries[:maxItems], nil
		}
		return feedEntries, nil
	}
}

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
			link := strings.Trim(line[2:], " \t")
			splits := strings.SplitN(link, " ", 2)
			if len(splits) == 2 && len(splits[1]) >= 10 {
				dateString := splits[1][:10]
				date, err := time.Parse("2006-01-02", dateString)
				if err != nil {
					continue
				}
				parsedUrl, err := url.Parse(splits[0])
				if err != nil {
					continue
				}
				if parsedUrl.Host == "" {
					// Is relative link
					parsedUrl.Host = baseUrl.Host
					parsedUrl.Path = path.Join(path.Dir(baseUrl.Path), parsedUrl.Path)
				}
				if time.Now().After(date) {
					title := strings.Trim(splits[1][10:], " -\t")
					fe := FeedEntry{title, parsedUrl, date, dateString, &gf}
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
