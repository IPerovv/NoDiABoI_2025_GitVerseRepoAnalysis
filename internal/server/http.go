package server

import (
	"context"
	"fmt"
	"encoding/json"
	"path/filepath"
	"net/http"
	"sync"
	"os"
	"time"

	"gitverse-analyser-service/internal/service"
	"gitverse-analyser-service/internal/config"
)

var (
	running bool
	mu      sync.Mutex
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/run", handleRun)
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/export", handleToCSVXLSX)
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


func handleToCSVXLSX(w http.ResponseWriter, r *http.Request) {
	outputDir := filepath.Join("dataset", "tables")

	if _, err := os.Stat(config.PathToJSONDataset); os.IsNotExist(err) {
		http.Error(w, "dataset file not found: " + config.PathToJSONDataset, http.StatusNotFound)
		return
	}

	file, err := os.Open(config.PathToJSONDataset)
	if err != nil {
		http.Error(w, "failed to open JSON: " + err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		http.Error(w, "failed to create output dir: " + err.Error(), http.StatusInternalServerError)
		return
	}

	csvPath, err := service.ConvertJSONToCSVXLSX(file.Name(), outputDir)
	if err != nil {
		http.Error(w, "failed to convert dataset to readable data: " + err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"filePath": csvPath,
	})
	fmt.Println("Exported dataset to files with name: %s", csvPath)
}
