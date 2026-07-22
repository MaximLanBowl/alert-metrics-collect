package main

import (
	"flag"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/agent"
)

func main() {
	addr := flag.String("a", "localhost:8080", "server listen address")
	reportInterval := flag.Duration("r", time.Second*10, "report timeout")
	pollInterval := flag.Duration("p", time.Second*2, "poll timeout")
	flag.Parse()

	baseURL := "http://" + *addr

	collector := agent.NewMemCollect(baseURL, *reportInterval, *pollInterval)
	collector.Run()
}
