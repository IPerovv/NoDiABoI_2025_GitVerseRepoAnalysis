package fetcher

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gitverse-analyser-service/internal/model"
	"gitverse-analyser-service/internal/storage"
)

func WorkerFunc(ctx context.Context, id int, jobs <-chan string, wg *sync.WaitGroup, rateLimiter <-chan time.Time) {
	defer wg.Done()
	for full := range jobs {
		<-rateLimiter
		u := fmt.Sprintf("https://gitverse.ru/sc/sbt/api/v1/repos/%s", full)
		u = strings.ReplaceAll(u, " ", "%20")

		var data model.RepoInfo
		err := safeGet(ctx, u, &data)
		if err != nil {
			fmt.Printf("worker %d: failed to fetch %s: %v\n", id, full, err)
			continue
		}

		storage.StoreMu.Lock()
		storage.RepoStore[data.ID] = data
		storage.StoreMu.Unlock()
	}
}
