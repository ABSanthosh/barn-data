package main

import (
	Util "BarnData/utils"
	"fmt"
	"path/filepath"

	"github.com/mmcdole/gofeed"
)

func main() {
	recommended := Util.ReadRecommended()

	fp := gofeed.NewParser()

	for topicName, topicItems := range recommended {

		var feedItems []Util.FeedItem
		for _, source := range topicItems {
			items, err := Util.FetchFeed(fp, source)
			if err != nil {
				fmt.Printf("Error fetching feed %s: %v\n", source.XmlUrl, err)
				continue
			}
			feedItems = append(feedItems, items...)
		}

		fileName := filepath.Join("today", topicName, topicName+".json")
		if err := Util.SaveJSONToFile(fileName, feedItems); err != nil {
			fmt.Println("Error writing JSON to file:", err)
			return
		}
		fmt.Println("Wrote", fileName)
	}
}
