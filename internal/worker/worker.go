package worker

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Worker struct {
	stopChan    chan struct{}
	requestsPtr *atomic.Int64
	getKeyCount func() int
}

func New(requestsPtr *atomic.Int64, getKeyCount func() int) *Worker {
	return &Worker{
		stopChan:    make(chan struct{}),
		requestsPtr: requestsPtr,
		getKeyCount: getKeyCount,
	}
}

func (w *Worker) Start() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			requests := w.requestsPtr.Load()
			keys := w.getKeyCount()
			fmt.Printf("Stats - Requests: %d, Keys: %d\n", requests, keys)
		case <-w.stopChan:
			fmt.Println("Worker stopped")
			return
		}
	}
}

func (w *Worker) Stop() {
	close(w.stopChan)
}
