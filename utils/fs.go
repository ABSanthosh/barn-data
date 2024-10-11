package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadRecommended reads the recommended data from the "recommended.json" file.
// It returns a Recommended struct and an error if any occurred during the process.
func ReadRecommended() Recommended {
	data, err := os.ReadFile("./utils/recommended.json")
	if err != nil {
		fmt.Println("error reading recommended.json:", err)
		os.Exit(1)
	}

	var recommended Recommended
	if err := json.Unmarshal(data, &recommended); err != nil {
		fmt.Println("error unmarshaling recommended data:", err)
		os.Exit(1)
	}

	return recommended
}

func ReadRssSource(filePath string) ([]FeedItem, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("error reading rss source file:", err)
		return nil, err
	}

	var feedItems []FeedItem
	if err := json.Unmarshal(data, &feedItems); err != nil {
		fmt.Println("error unmarshaling rss source data:", err)
		return nil, err
	}

	return feedItems, nil
}

// SaveJSONToFile saves the given data to the specified file.
func SaveJSONToFile(fileName string, data interface{}) error {
	feedItemJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("error marshaling data: %w", err)
		panic(err)
	}

	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		fmt.Println("error creating directories: %w", err)
		panic(err)
	}

	return os.WriteFile(fileName, feedItemJSON, 0644)
}
