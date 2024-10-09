package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

import Types "BarnData/utils"

func normalizeString(s string) string {
	// Unescape HTML entities
	s = html.UnescapeString(s)

	// Remove HTML tags
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s // Return original string if parsing fails
	}

	var extractText func(*html.Node) string
	extractText = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		var result string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			result += extractText(c)
		}
		return result
	}

	// Extract text content
	text := extractText(doc)

	// Remove extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

func fetchFeedWithRetry(ctx context.Context, fp *gofeed.Parser, source Types.Topic) ([]Types.FeedItem, error) {
	maxRetries := 3
	baseDelay := 5 * time.Second

	customClient := createCustomClient()
	customParser := gofeed.NewParser()
	customParser.Client = customClient
	customParser.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	for attempt := 0; attempt < maxRetries; attempt++ {
		feed, err := customParser.ParseURLWithContext(source.XmlUrl, ctx)
		if err != nil {
			if isTimeoutError(err) {
				delay := calculateBackoff(baseDelay, attempt)
				fmt.Printf("Timeout error for %s (attempt %d/%d). Retrying in %v...\n", source.XmlUrl, attempt+1, maxRetries, delay)
				time.Sleep(delay)
				continue
			} else if isRateLimitError(err) {
				delay := calculateBackoff(baseDelay, attempt)
				fmt.Printf("Rate limit hit for %s. Retrying in %v...\n", source.XmlUrl, delay)
				time.Sleep(delay)
				continue
			} else if isForbiddenError(err) {
				fmt.Printf("403 Forbidden error for %s. Trying alternative method...\n", source.XmlUrl)
				feed, err = fetchWithCustomRequest(ctx, customClient, source.XmlUrl)
				if err != nil {
					if isTimeoutError(err) {
						delay := calculateBackoff(baseDelay, attempt)
						fmt.Printf("Timeout error for %s (attempt %d/%d). Retrying in %v...\n", source.XmlUrl, attempt+1, maxRetries, delay)
						time.Sleep(delay)
						continue
					}
					return nil, fmt.Errorf("error fetching RSS feed %s: %w", source.XmlUrl, err)
				}
			} else {
				return nil, fmt.Errorf("error fetching RSS feed %s: %w", source.XmlUrl, err)
			}
		}

		var feedItems []Types.FeedItem
		for _, item := range feed.Items {
			var authorName string
			if len(item.Authors) > 0 {
				authorName = item.Authors[0].Name
			}
			feedItems = append(feedItems, Types.FeedItem{
				Title:       normalizeString(item.Title),
				Link:        item.Link,
				Description: normalizeString(item.Description),
				Author:      normalizeString(authorName),
				Published:   item.Published,
				Image:       extractImage(item),
				Source:      source.XmlUrl,
				Categories:  item.Categories,
			})
		}
		return feedItems, nil
	}
	return nil, fmt.Errorf("max retries reached for %s", source.XmlUrl)
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	if err.Error() == "Client.Timeout exceeded while awaiting headers" {
		return true
	}
	return false
}

func isRateLimitError(err error) bool {
	if httpErr, ok := err.(gofeed.HTTPError); ok {
		return httpErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

func isForbiddenError(err error) bool {
	if httpErr, ok := err.(gofeed.HTTPError); ok {
		return httpErr.StatusCode == http.StatusForbidden
	}
	return false
}

func fetchWithCustomRequest(ctx context.Context, client *http.Client, url string) (*gofeed.Feed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return gofeed.NewParser().Parse(resp.Body)
}

func calculateBackoff(baseDelay time.Duration, attempt int) time.Duration {
	backoff := baseDelay * time.Duration(1<<uint(attempt))
	jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
	return backoff + jitter
}

func extractImage(item *gofeed.Item) string {
	if len(item.Enclosures) > 0 {
		return item.Enclosures[0].URL
	}
	if item.Image != nil {
		return item.Image.URL
	}
	return ""
}

func createCustomClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: 60 * time.Second, // Increased timeout
		Jar:     jar,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

func main() {
	data, err := os.ReadFile("recommended.json")
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}

	var recommended Types.Recommended
	if err := json.Unmarshal(data, &recommended); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	fp := gofeed.NewParser()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent goroutines

	for topicName, topics := range recommended {
		wg.Add(1)
		go func(topicName string, topics []Types.Topic) {
			defer wg.Done()

			var feedItems []Types.FeedItem
			for _, source := range topics {
				semaphore <- struct{}{}                                                 // Acquire semaphore
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute) // Increased timeout
				items, err := fetchFeedWithRetry(ctx, fp, source)
				cancel()
				<-semaphore // Release semaphore

				if err != nil {
					fmt.Printf("Error fetching feed %s: %v\n", source.XmlUrl, err)
					continue
				}
				feedItems = append(feedItems, items...)
			}

			fileName := filepath.Join("today", topicName+".json")
			if err := saveJSONToFile(fileName, feedItems); err != nil {
				fmt.Println("Error writing JSON to file:", err)
				return
			}
			fmt.Println("============> Wrote", fileName)
		}(topicName, topics)
	}

	wg.Wait()
}

func saveJSONToFile(fileName string, data interface{}) error {
	feedItemJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return fmt.Errorf("error creating directories: %w", err)
	}

	return os.WriteFile(fileName, feedItemJSON, 0644)
}
