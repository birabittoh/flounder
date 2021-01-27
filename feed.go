package main

import (
	"bufio"
	"fmt"
	"github.com/mmcdole/gofeed"
	"os"
	"path"
)

const followingPath = "following.txt"
const followingFile = "following.gmi"

// TODO also get gemini, gemfeed

func getAllFeeds(user string) error {
	// Open file
	file, err := os.Open(path.Join(getUserDirectory(user), followingPath))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		feedURL := scanner.Text()
		// if scheme is gemini and filetype is gemini... gemtext
		// if scheme is gemini and filetype is xml/rss... fetch data  and parse
		fp := gofeed.NewParser()
		fp.ParseURL(feedURL)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	data := struct {
		DayContent []struct {
			date      string
			feed      *gofeed.Feed
			feedItems []gofeed.Item
		}
	}{}
	fmt.Println(data)
	// convert to gemini template
	// write template to file
	return nil
}
