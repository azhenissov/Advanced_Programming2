package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/azhenissov/Advanced_Programming2/internal/server"
	"github.com/azhenissov/Advanced_Programming2/internal/worker"
)

func main() {
	srv := server.New()

	w := worker.New(srv.GetRequestsPtr(), srv.GetKeyCount)
	go w.Start()

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: srv.Handler(),
	}

	go func() {
		fmt.Println("Server starting on port 8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutdown signal received")

	w.Stop()
	srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown error: %v\n", err)
	}

	fmt.Println("Server stopped gracefully")
}
