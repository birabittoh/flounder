package main

import (
	"bufio"
	"fmt"
	"git.sr.ht/~adnano/go-gemini"
	"github.com/mmcdole/gofeed"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"time"
)

const followingPath = "following.txt"
const followingFile = "following.gmi"

// TODO also get gemini, gemfeed

func feedsWorker() {
	log.Println("Starting feeds worker")
	for {
		users, err := getActiveUserNames()
		if err != nil {
			// Handle error somehow
			fmt.Println(err)
			continue
		}
		for _, user := range users {
			writeAllFeeds(user)
		}
		time.Sleep(time.Hour * 1)
	}
}

func writeAllFeeds(user string) error {
	// Open file
	file, err := os.Open(path.Join(getUserDirectory(user), followingPath))
	log.Println("Writing feeds for user " + user)
	defer file.Close()

	feedData := []*gofeed.Feed{}
	if err == nil {
		scanner := bufio.NewScanner(file)
		count := 1
		for scanner.Scan() {
			if count > 100 { // max number of lines
				break
			}
			count = count + 1
			feedURL := scanner.Text()
			parsed, err := url.Parse(feedURL)
			var feed *gofeed.Feed
			fp := gofeed.NewParser()
			if err != nil {
				log.Println("Invalid url " + feedURL)
			}
			if parsed.Scheme == "gemini" {
				client := gemini.Client{
					Timeout: 10 * time.Second,
				}
				res, err := client.Get(feedURL)
				defer res.Body.Close()
				if err != nil {
					log.Println(err)
					continue
				}
				if err != nil {
					log.Println(err)
					continue
				}
				feed, err = fp.Parse(res.Body)
				if err != nil {
					log.Println(err)
					continue
				}
			} else {
				// TODO if scheme is gemini and filetype is gemini... gemtext
				// TODO rate limit etc
				fp.Client = &http.Client{
					Timeout: 10 * time.Second,
				}
				feed, err = fp.ParseURL(feedURL)
				if err != nil {
					log.Println("Error getting feed " + feedURL)
					continue
				}
			}
			log.Println("Got feed data from " + feedURL)
			feedData = append(feedData, feed)
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	// Aggregate and sort by date
	type feedPlusItem struct {
		Feed     *gofeed.Feed
		FeedItem *gofeed.Item
		Date     string
	}
	data := struct {
		User      string
		FeedItems []feedPlusItem
	}{}
	data.User = user
	for _, feed := range feedData {
		for _, item := range feed.Items {
			if item.UpdatedParsed == nil {
				item.UpdatedParsed = item.PublishedParsed
			}
			if item.UpdatedParsed != nil {
				date := item.UpdatedParsed.Format("2006-01-02")
				data.FeedItems = append(data.FeedItems, feedPlusItem{feed, item, date})
			}
		}
	}
	sort.Slice(data.FeedItems, func(i, j int) bool {
		return data.FeedItems[i].FeedItem.UpdatedParsed.After(*data.FeedItems[j].FeedItem.UpdatedParsed)
	})
	maxItems := 100
	if len(data.FeedItems) > maxItems {
		data.FeedItems = data.FeedItems[:maxItems]
	}

	outputf, err := os.OpenFile(path.Join(getUserDirectory(user), followingFile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	err = gt.ExecuteTemplate(outputf, "following.gmi", data)
	if err != nil {
		return err
	}
	defer outputf.Close()
	// convert to gemini template
	// write template to file
	return nil
}
