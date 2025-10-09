package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"gitverse-analyser-service/internal/server"
	"gitverse-analyser-service/internal/storage"
)

func main() {
	ctx := context.Background()

	if err := storage.InitMongo(ctx); err != nil {
		fmt.Println("Mongo connection error:", err)
		os.Exit(1)
	}
	defer storage.CloseMongo(ctx)

	fmt.Println("Starting HTTP server on :8080 ...")
	srv := &http.Server{
		Addr:    ":8080",
		Handler: server.NewRouter(),
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}
