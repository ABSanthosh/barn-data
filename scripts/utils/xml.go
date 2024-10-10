package utils

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

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

func CustomClient() *http.Client {
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

func extractImage(item *gofeed.Item) string {
	if len(item.Enclosures) > 0 {
		return item.Enclosures[0].URL
	}
	if item.Image != nil {
		return item.Image.URL
	}
	return ""
}

// Types of errors
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

func calculateBackoff(baseDelay time.Duration, attempt int) time.Duration {
	backoff := baseDelay * time.Duration(1<<uint(attempt))
	jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
	return backoff + jitter
}

func customRequest(ctx context.Context, client *http.Client, url string) (*gofeed.Feed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
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

func FetchFeed(fp *gofeed.Parser, source Topic) ([]FeedItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fp.Client = CustomClient()
	fp.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"

	maxRetries := 3
	baseDelay := 5 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		feed, err := fp.ParseURLWithContext(source.XmlUrl, ctx)
		if err != nil {
			if isTimeoutError(err) {
				delay := calculateBackoff(baseDelay, attempt)
				fmt.Printf("> Timeout error: %v. Retrying in %v\n", source.XmlUrl, delay)
				time.Sleep(delay)
				continue
			} else if isRateLimitError(err) {
				delay := calculateBackoff(baseDelay, attempt)
				fmt.Printf("> Rate limit error: %v. Retrying in %v\n", source.XmlUrl, delay)
				time.Sleep(delay)
				continue
			} else if isForbiddenError(err) {
				fmt.Printf("> Forbidden error: %v\n", source.XmlUrl)
				feed, err = customRequest(ctx, fp.Client, source.XmlUrl)
				if err != nil {
					if isTimeoutError(err) {
						delay := calculateBackoff(baseDelay, attempt)
						fmt.Printf(">> Timeout error: %v. Retrying in %v\n", source.XmlUrl, delay)
						time.Sleep(delay)
						continue
					} else if isRateLimitError(err) {
						delay := calculateBackoff(baseDelay, attempt)
						fmt.Printf(">> Rate limit error: %v. Retrying in %v\n", source.XmlUrl, delay)
						time.Sleep(delay)
						continue
					} else {
						fmt.Printf(">> Error fetching feed %s: %v\n", source.XmlUrl, err)
						return nil, err
					}
				}
			} else {
				fmt.Printf("> Error fetching feed %s: %v\n", source.XmlUrl, err)
				return nil, err
			}
		}

		var feedItems []FeedItem
		for _, item := range feed.Items {

			authorName := ""
			if len(item.Authors) > 0 {
				authorName = item.Authors[0].Name
			}

			feedItems = append(feedItems, FeedItem{
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

	return nil, fmt.Errorf("> Max retries reached for %s", source.XmlUrl)
}
