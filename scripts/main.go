package main

import (
	Util "BarnData/utils"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/mmcdole/gofeed"
)

func main() {

	isFeed := flag.Bool("feed", false, "Run the feed command")
	isReadability := flag.Bool("readability", false, "Run the readability command")

	// Parse the command-line flags
	flag.Parse()

	if *isFeed {
		recommended := Util.ReadRecommended()

		fp := gofeed.NewParser()

		for topicName, topicItems := range recommended {

			var feedItems []Util.FeedItem
			for _, source := range topicItems {
				items, err := Util.FetchFeed(fp, source)
				if err != nil {
					Util.Chalk("Error fetching feed %s: %v\n", "red", source.XmlUrl, err)
					continue
				}
				feedItems = append(feedItems, items...)

				fmt.Println("")
			}

			fileName := filepath.Join("today", topicName, topicName+".json")
			if err := Util.SaveJSONToFile(fileName, feedItems); err != nil {
				fmt.Println("Error writing JSON to file:", err)
				return
			}
			fmt.Println("Wrote", fileName)
		}
	} else if *isReadability {
		// Run the readability command
	} else {
		fmt.Println("Please specify a command: --feed or --readability")
	}
}
