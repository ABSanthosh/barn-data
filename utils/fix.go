package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"strings"
)

func FixFeeds() {
	topicFolders, err := filepath.Glob("today/*")
	removeCount := 0
	if err != nil {
		fmt.Println("error reading topic folders:", err)
		os.Exit(1)
	}

	for _, topicFolder := range topicFolders {
		removeCount = 0
		matches, err := filepath.Glob(filepath.Join(topicFolder, "*.json"))
		if err != nil {
			fmt.Println("error reading files in topic folder:", err)
			os.Exit(1)
		}
		if len(matches) == 0 {
			fmt.Println("no json files found in", topicFolder)
			continue
		}

		var topicJson []FeedItem
		topicJsonFile, _ := os.Open(matches[0])
		defer topicJsonFile.Close()
		reader := bufio.NewReader(topicJsonFile)
		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(&topicJson); err != nil {
			fmt.Println("error decoding topic json:", err)
			os.Exit(1)
		}
		readabilityFiles, _ := filepath.Glob(filepath.Join(topicFolder, "readability", "*.json"))
		readabilityFiles = lop.Map(readabilityFiles, func(file string, _ int) string {
			return strings.Split(filepath.Base(file), ".")[0]
		})

		for index, file := range topicJson {
			if !lo.Contains(readabilityFiles, file.ID) {
				topicJson = append(topicJson[:index], topicJson[index+1:]...)
				removeCount++
			}
		}

		if removeCount > 0 {
			fmt.Println("removed", removeCount, "items from", topicFolder)
			SaveJSONToFile(matches[0], topicJson)
		} else {
			fmt.Println("no items removed from", topicFolder)
		}
	}
}
