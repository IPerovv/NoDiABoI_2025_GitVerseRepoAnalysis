package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RepoInfo struct {
	ID             int64     `json:"id"`
	FullName       string    `json:"full_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Archived       bool      `json:"archived"`
	StarsCount     int       `json:"stars_count"`
	Size           int       `json:"size"`
	ReleaseCounter int       `json:"release_counter"`
	TagCount       int       `json:"tag_count"`
}

var (
	repoStore   = make(map[int64]RepoInfo)
	storeMu     sync.Mutex
	httpClient  *http.Client
	userAgent   = "gitverse-scraper/1.0"
	maxWorkers  = 40
	ratePerSec  = 100
	retryMax    = 6
	pageLimit   = 50
	searchLimit = 30
)

func initHttpClient() {
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
}

func safeGet(ctx context.Context, fullURL string, target interface{}) error {
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt < retryMax; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		req.Header.Set("User-Agent", userAgent)
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

func fetchAllRepoNames(ctx context.Context) ([]string, error) {
	const pageLimit = 50

	total, err := fetchTotalRepos(ctx)
	if err != nil {
		return nil, err
	}
	totalPages := (total + pageLimit - 1) / pageLimit

	names := make([]string, 0, total)
	pages := make(chan int, totalPages)
	for i := 1; i <= totalPages; i++ {
		pages <- i
	}
	close(pages)

	var mu sync.Mutex
	var wg sync.WaitGroup
	maxWorkers := 10

	wg.Add(maxWorkers)
	for w := 0; w < maxWorkers; w++ {
		go func() {
			defer wg.Done()
			for page := range pages {
				repos, err := fetchRepoPage(ctx, page, pageLimit)
				if err != nil {
					fmt.Println("Error fetching page", page, ":", err)
					continue
				}
				mu.Lock()
				names = append(names, repos...)
				mu.Unlock()
			}
		}()
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	done := make(chan struct{})
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

	return names, nil
}

func fetchTotalRepos(ctx context.Context) (int, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://gitverse.ru/sc/sbt/api/v1/repos/search?page=1&limit=1", nil)
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

func fetchRepoPage(ctx context.Context, page, limit int) ([]string, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		url := fmt.Sprintf("https://gitverse.ru/sc/sbt/api/v1/repos/search?page=%d&limit=%d", page, limit)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				var result struct {
					Data []struct {
						FullName string `json:"fullName"`
					} `json:"data"`
				}
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

func workerFunc(ctx context.Context, id int, jobs <-chan string, wg *sync.WaitGroup, rateLimiter <-chan time.Time) {
	defer wg.Done()
	for full := range jobs {
		<-rateLimiter
		u := fmt.Sprintf("https://gitverse.ru/sc/sbt/api/v1/repos/%s", full)
		u = strings.ReplaceAll(u, " ", "%20")

		var data struct {
			ID             int64     `json:"id"`
			Name           string    `json:"name"`
			FullName       string    `json:"full_name"`
			CreatedAt      time.Time `json:"created_at"`
			UpdatedAt      time.Time `json:"updated_at"`
			Archived       bool      `json:"archived"`
			StarsCount     int       `json:"stars_count"`
			Size           int       `json:"size"`
			ReleaseCounter int       `json:"release_counter"`
			TagCount       int       `json:"tag_count"`
		}

		err := safeGet(ctx, u, &data)
		if err != nil {
			fmt.Printf("worker %d: failed to fetch %s: %v\n", id, full, err)
			continue
		}

		r := RepoInfo{
			ID:             data.ID,
			FullName:       data.FullName,
			CreatedAt:      data.CreatedAt,
			UpdatedAt:      data.UpdatedAt,
			Archived:       data.Archived,
			StarsCount:     data.StarsCount,
			Size:           data.Size,
			ReleaseCounter: data.ReleaseCounter,
			TagCount:       data.TagCount,
		}

		storeMu.Lock()
		repoStore[r.ID] = r
		storeMu.Unlock()
	}
}

func main() {
	initHttpClient()

	ctx := context.Background()

	fmt.Println("Fetching list of all repository names...")
	names, err := fetchAllRepoNames(ctx)
	if err != nil {
		fmt.Println("Error fetching repo list:", err)
		return
	}
	fmt.Printf("Got %d repo names\n", len(names))

	jobs := make(chan string, len(names))
	for _, n := range names {
		jobs <- n
	}
	close(jobs)

	interval := time.Second / time.Duration(ratePerSec)
	if interval <= 0 {
		interval = time.Millisecond
	}
	rateTicker := time.NewTicker(interval)
	defer rateTicker.Stop()

	rateChan := make(chan time.Time)
	go func() {
		for t := range rateTicker.C {
			select {
			case rateChan <- t:
			default:
			}
		}
	}()

	var wg sync.WaitGroup
	start := time.Now()
	fmt.Println("Starting workers...")
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go workerFunc(ctx, i, jobs, &wg, rateChan)
	}

	var mu sync.Mutex
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			mu.Lock()
			fmt.Printf("Progress: fetched %d/%d repos...\n", len(repoStore), len(names))
			mu.Unlock()
		}
	}()

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("Done. Fetched %d repos in %s\n", len(repoStore), elapsed)

	printTopStars(10)
}

func printTopStars(n int) {
	type pair struct {
		Name  string
		Stars int
	}
	list := make([]pair, 0, len(repoStore))
	storeMu.Lock()
	for _, r := range repoStore {
		list = append(list, pair{r.FullName, r.StarsCount})
	}
	storeMu.Unlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i].Stars > list[j].Stars
	})

	fmt.Printf("Top %d repos by stars:\n", n)
	for i := 0; i < n && i < len(list); i++ {
		fmt.Printf("%d) %s — %d stars\n", i+1, list[i].Name, list[i].Stars)
	}
}
