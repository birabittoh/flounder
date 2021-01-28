package main

import (
	"bufio"
	"fmt"
	"github.com/mmcdole/gofeed"
	"log"
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
		time.Sleep(time.Hour * 1)
		users, err := getActiveUserNames()
		if err != nil {
			// Handle error somehow
			fmt.Println(err)
			continue
		}
		for _, user := range users {
			writeAllFeeds(user)
		}
	}
}

func writeAllFeeds(user string) error {
	// Open file
	file, err := os.Open(path.Join(getUserDirectory(user), followingPath))
	if err != nil {
		if os.IsNotExist(err) {
			// TODO
			return nil
		}
		return err
	}
	log.Println("Writing feeds for user " + user)
	defer file.Close()

	feedData := []*gofeed.Feed{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		feedURL := scanner.Text()
		// TODO if scheme is gemini and filetype is gemini... gemtext
		// TODO if scheme is gemini and filetype is xml/rss... fetch data  and parse
		// TODO rate limit etc
		fp := gofeed.NewParser()
		feed, err := fp.ParseURL(feedURL)
		if err != nil {
			log.Println("Error getting feed " + feedURL)
		} else {
			log.Println("Got feed data from " + feedURL)
			feedData = append(feedData, feed)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
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
