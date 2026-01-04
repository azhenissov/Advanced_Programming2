package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/azhenissov/Advanced_Programming2/internal/server"
	"github.com/azhenissov/Advanced_Programming2/internal/store"
	"github.com/azhenissov/Advanced_Programming2/internal/worker"

)

func main() {
	dataStore := store.NewStore[string, string]()
	srv := server.NewServer(dataStore)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	stopWorker := make(chan struct{})
	go worker.StartWorker(srv, stopWorker)

	go func() {
		log.Println("Starting server on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :8080: %v\n", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down server...")
	close(stopWorker)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpServer.Shutdown(ctx)
	log.Println("Server gracefully stopped")
}
