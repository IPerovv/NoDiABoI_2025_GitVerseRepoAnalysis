package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"gitverse-analyser-service/internal/service"
)

var (
	running bool
	mu      sync.Mutex
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/run", handleRun)
	mux.HandleFunc("/status", handleStatus)
	return mux
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	if running {
		mu.Unlock()
		http.Error(w, "Already running", http.StatusConflict)
		return
	}
	running = true
	mu.Unlock()

	go func() {
		defer func() {
			mu.Lock()
			running = false
			mu.Unlock()
		}()
		ctx := context.Background()
		service.RunFullScrape(ctx)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"running": running,
	})
}
