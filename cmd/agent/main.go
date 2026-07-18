package main

import (
	"github.com/MaximLanBowl/alert-metrics-collect/internal/agent"
)

func main() {
	collector := agent.NewMemCollect("http://localhost:8080")
	collector.Run()
}
