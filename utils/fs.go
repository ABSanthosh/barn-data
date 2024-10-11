package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	bufferSize = 4096 // 4KB buffer
	batchSize  = 100  // Number of items to process in a batch
)

// ReadRecommended reads the recommended data from the "recommended.json" file.
// It returns a Recommended struct and an error if any occurred during the process.
func ReadRecommended() Recommended {
	file, err := os.Open("./utils/recommended.json")
	if err != nil {
		fmt.Println("error opening recommended.json:", err)
		os.Exit(1)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	decoder := json.NewDecoder(reader)

	var recommended Recommended
	if err := decoder.Decode(&recommended); err != nil {
		fmt.Println("error decoding recommended data:", err)
		os.Exit(1)
	}

	return recommended
}

func ReadRssSource(filePath string) ([]FeedItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("error opening rss source file:", err)
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	decoder := json.NewDecoder(reader)

	var feedItems []FeedItem
	if err := decoder.Decode(&feedItems); err != nil {
		fmt.Println("error decoding rss source data:", err)
		return nil, err
	}

	return feedItems, nil
}

// SaveJSONToFile saves the given data to the specified file.
func SaveJSONToFile(fileName string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return fmt.Errorf("error creating directories: %w", err)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, bufferSize)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error encoding data: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("error flushing writer: %w", err)
	}

	return nil
}

// BatchSaveJSONToFiles saves multiple JSON files in batches
func BatchSaveJSONToFiles(fileDataPairs []struct {
	FileName string
	Data     interface{}
}) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(fileDataPairs))

	for i := 0; i < len(fileDataPairs); i += batchSize {
		end := i + batchSize
		if end > len(fileDataPairs) {
			end = len(fileDataPairs)
		}

		wg.Add(1)
		go func(batch []struct {
			FileName string
			Data     interface{}
		}) {
			defer wg.Done()
			for _, pair := range batch {
				if err := SaveJSONToFile(pair.FileName, pair.Data); err != nil {
					errChan <- fmt.Errorf("error saving file %s: %w", pair.FileName, err)
					return
				}
			}
		}(fileDataPairs[i:end])
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
