package worker

import (
	"fmt"
	"time"
)

type StatsProvider interface {
	RequestCount() uint64
	KeyCount() int
}

func StartWorker(sp StatsProvider, stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf(
				"[WORKER] requests=%d keys=%d\n",
				sp.RequestCount(),
				sp.KeyCount(),
			)
		case <-stop:
			fmt.Println("[WORKER] stopped")
			return
		}
	}
}
