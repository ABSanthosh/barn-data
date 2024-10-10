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
