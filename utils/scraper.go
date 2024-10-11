package utils

import (
	"context"
	"crypto/tls"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:90.0) Gecko/20100101 Firefox/90.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
}

func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// createClient creates and returns an HTTP client with custom settings.
// It sets a timeout of 60 seconds for the client, uses a cookie jar for managing cookies,
// and configures the transport settings for the client.
// The transport settings include a dial timeout of 30 seconds, a keep-alive duration of 30 seconds,
// a TLS handshake timeout of 15 seconds, a response header timeout of 30 seconds,
// an expect continue timeout of 1 second, and an insecure skip verify option for the TLS client configuration.
// The function returns the created HTTP client.
func createClient() *http.Client {
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
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// handleRateLimit handles rate limiting by waiting for the specified duration before retrying.
// If the retryAfter duration is not valid, it defaults to 60 seconds.
//
// Parameters:
//   - retryAfter: The duration to wait before retrying, in the format of a string.
//
// Example:
//
//	handleRateLimit("10s") // Rate limited. Waiting for 10s before retrying...
func handleRateLimit(retryAfter string) {
	duration, err := time.ParseDuration(retryAfter)
	if err != nil {
		duration = 60 * time.Second
	}
	Chalk("↳ Rate limited. Waiting for %v before retrying...\n", "cyan", duration)
	time.Sleep(duration)
}

// calculateBackoff calculates the backoff duration for retry attempts.
// It takes the attempt number, base delay, and maximum delay as input parameters.
// The backoff duration is calculated using an exponential backoff algorithm.
// The backoff duration is calculated as the base delay multiplied by 2 raised to the power of the attempt number.
// A random jitter is added to the backoff duration to introduce some randomness.
// The resulting backoff duration is capped between the base delay and the maximum delay.
// The calculated backoff duration is returned as a time.Duration value.
func calculateBackoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	backoff := float64(baseDelay) * math.Pow(2, float64(attempt))
	jitter := rand.Float64() * float64(baseDelay)
	return time.Duration(math.Max(math.Min(backoff+jitter, float64(maxDelay)), float64(baseDelay)))
}

func fetchWebpage(ctx context.Context, url string, maxRetries int) (string, error) {
	client := createClient()
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Create a new context with a timeout for each request
		reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
		if err != nil {
			return "", err
		}

		req.Header.Set("User-Agent", getRandomUserAgent())
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

		resp, err := client.Do(req)
		if err != nil {
			if reqCtx.Err() == context.DeadlineExceeded {
				Chalk("Request timed out, retrying...\n", "cyan")
			} else {
				backoffDuration := calculateBackoff(i, baseDelay, maxDelay)
				Chalk("Request failed, retrying in %v...\n", "cyan", backoffDuration)
				time.Sleep(backoffDuration)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			handleRateLimit(resp.Header.Get("Retry-After"))
		} else {
			backoffDuration := calculateBackoff(i, baseDelay, maxDelay)
			Chalk("↳ Received status code %d, retrying in %v...\n", "cyan", resp.StatusCode, backoffDuration)
			time.Sleep(backoffDuration)
		}
	}

	Chalk("↳ Failed to fetch webpage after %d retries\n", "red", maxRetries)
	return "", nil
}

// isMediaFile checks if the URL points to a media file (like mp3 or mp4).
func isMediaFile(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Get the file extension from the URL path
	ext := strings.ToLower(filepath.Ext(parsedURL.Path))

	// List of supported media file extensions
	supportedExtensions := []string{".mp3", ".mp4", ".wav", ".avi", ".mov", ".mkv"}

	// Check if the extension is in the supported list
	for _, supportedExt := range supportedExtensions {
		if ext == supportedExt {
			return true
		}
	}

	return false
}

func FetchPage(url string) (string, error) {
	maxRetries := 3
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for i := 0; i < maxRetries; i++ {
		content, err := fetchWebpage(ctx, url, 3)
		if err == nil {
			return content, nil
		}

		Chalk("↳ Attempt %d failed: %v\n", "Red", i+1, err)

		if i < maxRetries-1 {
			backoffDuration := calculateBackoff(i, 1*time.Second, 60*time.Second)
			Chalk("↳ Waiting for %v before next attempt...\n", "cyan", backoffDuration)
			time.Sleep(backoffDuration)
		}
	}

	Chalk("↳ failed to fetch webpage after %d retries \n", "red", maxRetries)
	return "", nil
}
