package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"gitverse-analyser-service/internal/config"
	"gitverse-analyser-service/internal/fetcher"
	"gitverse-analyser-service/internal/storage"
	"gitverse-analyser-service/internal/util"
)

func RunFullScrape(ctx context.Context) {
	fetcher.InitHttpClient()

	fmt.Println("Fetching list of all repository names...")
	names, err := fetcher.FetchAllRepoNames(ctx)
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

	rateChan := util.NewRateLimiter(config.RatePerSecond)

	var wg sync.WaitGroup
	start := time.Now()
	fmt.Println("Starting workers...")

	for i := 0; i < config.MaxWorkers; i++ {
		wg.Add(1)
		go fetcher.WorkerFunc(ctx, i, jobs, &wg, rateChan)
	}

	ticker := time.NewTicker(config.ProgressPeriod)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			storage.StoreMu.Lock()
			fmt.Printf("Progress: fetched %d/%d repos...\n", len(storage.RepoStore), len(names))
			storage.StoreMu.Unlock()
		}
	}()

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("Done. Fetched %d repos in %s\n", len(storage.RepoStore), elapsed)

	if err := storage.SaveAllToMongo(ctx); err != nil {
		fmt.Println("Error saving to Mongo:", err)
	}

	printTopStars(config.TopStarsCount)
}

func printTopStars(n int) {
	type pair struct {
		Name  string
		Stars int
	}
	list := make([]pair, 0, len(storage.RepoStore))
	storage.StoreMu.Lock()
	for _, r := range storage.RepoStore {
		list = append(list, pair{r.FullName, r.StarsCount})
	}
	storage.StoreMu.Unlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i].Stars > list[j].Stars
	})

	fmt.Printf("Top %d repos by stars:\n", n)
	for i := 0; i < n && i < len(list); i++ {
		fmt.Printf("%d) %s — %d stars\n", i+1, list[i].Name, list[i].Stars)
	}
}
