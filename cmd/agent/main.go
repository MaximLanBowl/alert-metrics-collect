package main

import (
	"flag"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/agent"
)

func main() {
	addr := flag.String("a", "localhost:8080", "server listen address")
	reportInterval := flag.Int("r", 10, "report timeout")
	pollInterval := flag.Int("p", 2, "poll timeout")

	flag.Parse()

	baseURL := "http://" + *addr
	rVal := time.Duration(*reportInterval) * time.Second
	pVal := time.Duration(*pollInterval) * time.Second

	collector := agent.NewMemCollect(baseURL, rVal, pVal)
	collector.Run()
}
