package main

import (
	Util "BarnData/utils"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mmcdole/gofeed"
)

func main() {
	isFeed := flag.Bool("feed", false, "Run the feed command")
	isReadability := flag.Bool("readability", false, "Run the readability command")
	isFix := flag.Bool("fix", false, "Run the fix command")
	flag.Parse()

	if *isFeed {
		processFeed()
	} else if *isReadability {
		processReadability()
	} else if *isFix {
		Util.FixFeeds()
	} else {
		fmt.Println("Please specify a command: --feed or --readability")
	}
}

func processFeed() {
	recommended := Util.ReadRecommended()
	fp := gofeed.NewParser()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent operations

	for topicName, topicItems := range recommended {
		wg.Add(1)
		go func(topicName string, topicItems []Util.Topic) {
			defer wg.Done()
			var feedItems []Util.FeedItem
			var mu sync.Mutex

			var itemWg sync.WaitGroup
			for _, source := range topicItems {
				itemWg.Add(1)
				go func(source Util.Topic) {
					defer itemWg.Done()
					semaphore <- struct{}{}        // Acquire semaphore
					defer func() { <-semaphore }() // Release semaphore

					items, err := Util.FetchFeed(fp, source)
					if err != nil {
						Util.Chalk("Error fetching feed %s: %v\n", "red", source.XmlUrl, err)
						return
					}

					mu.Lock()
					feedItems = append(feedItems, items...)
					mu.Unlock()
				}(source)
			}
			itemWg.Wait()

			fileName := filepath.Join("today", topicName, topicName+".json")
			if err := Util.SaveJSONToFile(fileName, feedItems); err != nil {
				fmt.Println("Error writing JSON to file:", err)
				return
			}
			fmt.Println("Wrote", fileName)
		}(topicName, topicItems)
	}
	wg.Wait()
}

func processReadability() {
	topicFolders, err := filepath.Glob("today/*")
	if err != nil {
		fmt.Println("Error reading topic folders:", err)
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent operations

	for _, topicFolder := range topicFolders {
		wg.Add(1)
		go func(topicFolder string) {
			defer wg.Done()
			fileName := filepath.Join(topicFolder, strings.Split(topicFolder, "/")[1]+".json")
			feedItems, _ := Util.ReadRssSource(fileName)

			var itemWg sync.WaitGroup
			for _, feedItem := range feedItems {
				itemWg.Add(1)
				go func(feedItem Util.FeedItem) {
					defer itemWg.Done()
					semaphore <- struct{}{}        // Acquire semaphore
					defer func() { <-semaphore }() // Release semaphore

					savePath := filepath.Join(topicFolder, "readability", feedItem.ID+".json")
					readability, err := Util.FetchReadability(feedItem)
					if err != nil {
						Util.Chalk("Error fetching readability for %s: %v\n", "red", feedItem.Link, err)
						return
					}
					Util.SaveJSONToFile(savePath, readability)
				}(feedItem)
			}
			itemWg.Wait()

			Util.Chalk("Wrote readability for %s\n", "green", topicFolder)
		}(topicFolder)
	}
	wg.Wait()
}
