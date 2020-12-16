// Parses Gemfeed according to the companion spec: gemini://gemini.circumlunar.space/docs/companion/subscription.gmi
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Gemfeed struct {
	Title   string
	Entries []*FeedEntry
}

type FeedEntry struct {
	Title      string
	Url        string
	Date       time.Time
	FeedTitle  string
	DateString string
}

// TODO definitely cache this function -- it reads EVERY gemini file on flounder.
func getAllGemfeedEntries() ([]*FeedEntry, error) {
	var feedEntries []*FeedEntry
	err := filepath.Walk(c.FilesDirectory, func(thepath string, info os.FileInfo, err error) error {
		if isGemini(info.Name()) {
			f, err := os.Open(thepath)
			feed, err := ParseGemfeed(f)
			if err == nil {
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
		return feedEntries, nil
	}
}

// Parsed Gemfeed text Returns error if not a gemfeed
// Doesn't sort output
// Doesn't get posts dated in the future
func ParseGemfeed(text io.Reader) (*Gemfeed, error) {
	scanner := bufio.NewScanner(text)
	gf := Gemfeed{}
	for scanner.Scan() {
		line := scanner.Text()
		if gf.Title == "" && strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "##") {
			gf.Title = strings.Trim(line[1:], " \t")
		} else if strings.HasPrefix(line, "=>") {
			link := strings.Trim(line[2:], " \t")
			splits := strings.SplitN(link, " ", 2)
			if len(splits) == 2 && len(splits[1]) >= 10 {
				dateString := splits[1][:10]
				date, err := time.Parse("2006-01-02", dateString)
				if err == nil && time.Now().After(date) {
					title := strings.Trim(splits[1][10:], " -\t")
					fe := FeedEntry{title, splits[0], date, gf.Title, dateString}
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
