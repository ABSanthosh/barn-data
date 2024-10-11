package utils

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
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

func extractImage(item *gofeed.Item) string {
	if len(item.Enclosures) > 0 {
		return item.Enclosures[0].URL
	}
	if item.Image != nil {
		return item.Image.URL
	}
	return ""
}

func FetchFeed(fp *gofeed.Parser, source Topic) ([]FeedItem, error) {
	Chalk("Fetching feed %s...\n", "magenta", source.XmlUrl)
	feedContent, err := FetchPage(source.XmlUrl)
	if err != nil {
		return nil, err
	}

	feed, err := fp.ParseString(feedContent)
	if err != nil {
		return nil, err
	}

	feedItems := make([]FeedItem, 0)
	for i, item := range feed.Items {
		// Its only 2 because we have 10 concurrent workers
		// and it'll add up to 20 items
		if i >= 2 {
			break
		}

		authorName := ""
		if len(item.Authors) > 0 {
			authorName = item.Authors[0].Name
		}

		feedItems = append(feedItems, FeedItem{
			ID:          uuid.New().String(),
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

	Chalk("Fetched %d items from %s\n", "green", len(feedItems), source.XmlUrl)

	return feedItems, nil
}
