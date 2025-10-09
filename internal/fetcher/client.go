package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitverse-analyser-service/internal/config"
)

var httpClient *http.Client

func InitHttpClient() {
	httpClient = &http.Client{
		Timeout: config.HTTPTimeout,
	}
}

func safeGet(ctx context.Context, fullURL string, target interface{}) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		req.Header.Set("User-Agent", config.UserAgent)
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp)
			resp.Body.Close()
			if retryAfter > 0 {
				fmt.Printf("429 received for %s — sleeping %s\n", fullURL, retryAfter)
				time.Sleep(retryAfter)
			} else {
				time.Sleep(backoff)
				backoff *= 2
			}
			lastErr = fmt.Errorf("429 Too Many Requests")
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			time.Sleep(backoff)
			backoff *= 2
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode != 200 {
			var b strings.Builder
			buf := make([]byte, 1024)
			for {
				n, _ := resp.Body.Read(buf)
				if n == 0 {
					break
				}
				b.Write(buf[:n])
				if b.Len() > 4096 {
					break
				}
			}
			resp.Body.Close()
			return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, b.String())
		}

		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(target); err != nil {
			resp.Body.Close()
			lastErr = err
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		resp.Body.Close()
		return nil
	}
	return fmt.Errorf("failed after retries: %w", lastErr)
}

func parseRetryAfter(resp *http.Response) time.Duration {
	if s := resp.Header.Get("GitVerse-RateLimit-Retry-After"); s != "" {
		if sec, err := strconv.Atoi(s); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	if s := resp.Header.Get("Retry-After"); s != "" {
		if sec, err := strconv.Atoi(s); err == nil {
			return time.Duration(sec) * time.Second
		}
		if t, err := http.ParseTime(s); err == nil {
			return time.Until(t)
		}
	}
	if s := resp.Header.Get("Gitverse-Ratelimit-Reset"); s != "" {
		if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
			resetTime := time.Unix(ts, 0)
			return time.Until(resetTime) + (1 * time.Second)
		}
	}
	return 0
}
