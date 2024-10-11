package utils

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"

	readability "github.com/go-shiori/go-readability"
)

func readingTime(length int) string {
	enCPM := 987
	variance := 118
	charactersPerMinuteLow := enCPM - variance
	charactersPerMinuteHigh := enCPM + variance

	durationLow := int(math.Ceil(float64(length) / float64(charactersPerMinuteLow)))
	durationHigh := int(math.Ceil(float64(length) / float64(charactersPerMinuteHigh)))
	duration := int(math.Round(float64(durationLow+durationHigh) / 2))

	if duration > 1 {
		return fmt.Sprintf("%d min read", duration)
	}
	return "Less than 1 min"
}

func cleanText(text string) string {
	// Replace multiple newlines with a single newline
	reNewline := regexp.MustCompile(`\n{2,}`)
	text = reNewline.ReplaceAllString(text, "\n")

	// Replace multiple spaces with a single space
	reSpaces := regexp.MustCompile(`\s+`)
	text = reSpaces.ReplaceAllString(text, " ")

	// Trim leading and trailing whitespace
	return strings.TrimSpace(text)
}

func FetchReadability(feedItem FeedItem) (ReadabilityItem, error) {

	if isMediaFile(feedItem.Link) {
		return ReadabilityItem{}, fmt.Errorf("media file")
	}

	webpage, err := FetchPage(feedItem.Link)
	if err == nil {
		Chalk("Fetching readability for %s\n", "green", feedItem.Link)
	}

	parsedURL, _ := url.Parse(feedItem.Link)

	article, err := readability.FromReader(strings.NewReader(webpage), parsedURL)
	if err != nil {
		return ReadabilityItem{}, err
	}

	return ReadabilityItem{
		FeedItem:      &feedItem,
		Image:         article.Image,
		Byline:        article.Byline,
		Length:        article.Length,
		Excerpt:       article.Excerpt,
		Favicon:       article.Favicon,
		HTML:          article.Content,
		SiteName:      article.SiteName,
		ModifiedTime:  article.ModifiedTime,
		PublishedTime: article.PublishedTime,
		ReadingTime:   readingTime(article.Length),
		Text:          cleanText(article.TextContent),
	}, nil
}
