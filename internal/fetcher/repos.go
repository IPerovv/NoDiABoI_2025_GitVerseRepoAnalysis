package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"gitverse-analyser-service/internal/config"
)

func FetchTotalRepos(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s%s?page=1&limit=1", config.BaseURL, config.SearchPath)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Total, nil
}

func FetchAllRepoNames(ctx context.Context) ([]string, error) {
	total := 100
	totalPages := (total + config.PageLimit - 1) / config.PageLimit

	names := make([]string, 0, total)
	pages := make(chan int, totalPages)
	for i := 1; i <= totalPages; i++ {
		pages <- i
	}
	close(pages)

	var mu sync.Mutex
	var wg sync.WaitGroup
	retryCh := make(chan int, totalPages/2)
	done := make(chan struct{})

	wg.Add(config.MaxWorkers)
	for w := 0; w < config.MaxWorkers; w++ {
		go func() {
			defer wg.Done()
			for page := range pages {
				repos, err := fetchRepoPage(ctx, page, config.PageLimit)
				if err != nil {
					fmt.Printf("Error fetching page %d: %v (will retry)\n", page, err)
					retryCh <- page
					continue
				}
				mu.Lock()
				names = append(names, repos...)
				mu.Unlock()
			}
		}()
	}

	ticker := time.NewTicker(config.ProgressPeriod)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				fmt.Printf("Progress (fetchAllRepoNames): fetched %d/%d repos...\n", len(names), total)
				mu.Unlock()
			case <-done:
				return
			}
		}
	}()

	wg.Wait()
	close(done)
	close(retryCh)

	if len(retryCh) > 0 {
		fmt.Printf("Retrying %d failed pages...\n", len(retryCh))
		for p := range retryCh {
			repos, err := fetchRepoPage(ctx, p, config.PageLimit)
			if err != nil {
				fmt.Printf("Final fail page %d: %v\n", p, err)
				continue
			}
			mu.Lock()
			names = append(names, repos...)
			mu.Unlock()
		}
	}

	return names, nil
}

func fetchRepoPage(ctx context.Context, page, limit int) ([]string, error) {
	url := fmt.Sprintf("%s%s?page=%d&limit=%d", config.BaseURL, config.SearchPath, page, limit)
	var result struct {
		Data []struct {
			FullName string `json:"fullName"`
		} `json:"data"`
	}

	var lastErr error
	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		req.Header.Set("User-Agent", config.UserAgent)
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					lastErr = err
				} else {
					names := make([]string, len(result.Data))
					for i, r := range result.Data {
						names[i] = r.FullName
					}
					return names, nil
				}
			} else if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("status %d", resp.StatusCode)
			} else {
				lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
			}
		}
		sleep := time.Duration((1<<attempt)*500) * time.Millisecond
		sleep += time.Duration(rand.Intn(300)) * time.Millisecond
		time.Sleep(sleep)
	}
	return nil, lastErr
}
