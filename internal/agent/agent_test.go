package agent

import (
	"compress/gzip"
	"fmt"
	"io"
	"testing"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/assert"
)

func TestMemCollect(t *testing.T) {
	m := NewMemCollect(config.AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		SecretKey:      "",
		RateLimit:      30,
	})

	m.collect()

	wantMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction",
		"GCSys", "HeapAlloc", "HeapIdle", "HeapInuse",
		"HeapObjects", "HeapReleased", "HeapSys",
		"LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC",
		"NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs", "StackInuse",
		"StackSys", "Sys", "TotalAlloc", "RandomValue",
	}

	for _, metric := range wantMetrics {
		if _, ok := m.gauges[metric]; !ok {
			t.Errorf("metric %s not found", metric)
		}
	}

	if len(m.gauges) != len(wantMetrics) {
		t.Errorf("expected %d metrics, got %d", len(wantMetrics), len(m.gauges))
	}
}

func TestCollectCPU(t *testing.T) {
	m := NewMemCollect(config.AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		SecretKey:      "",
		RateLimit:      30,
	})

	percentCPU, err := cpu.Percent(0, true)
	if err != nil {
		t.Errorf("error getting CPU percent: %v", err)
	}
	numCPU := len(percentCPU)

	m.collectMemCPU()

	memory := []string{
		"TotalMemory", "FreeMemory",
	}
	for _, mt := range memory {
		if _, ok := m.gauges[mt]; !ok {
			t.Errorf("metric %s not found", mt)
		}
	}

	for i := 1; i <= numCPU; i++ {
		name := fmt.Sprintf("CPUutilization%d", i)
		if _, ok := m.gauges[name]; !ok {
			t.Errorf("metric %s not found", name)
		}
	}

	expectedTotal := len(memory) + numCPU

	if len(m.gauges) != expectedTotal {
		t.Errorf("expected %d metrics, got %d", expectedTotal, len(m.gauges))
	}

	t.Log("CPU AND Memory Gauges:", m.gauges)
}

func TestPollCountIncrements(t *testing.T) {
	m := NewMemCollect(config.AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		SecretKey:      "",
		RateLimit:      30,
	})

	m.collect()
	if got := m.counters["PollCount"]; got != 1 {
		t.Errorf("after 1st collect: expected PollCount=1, got %d", got)
	}

	m.collect()
	if got := m.counters["PollCount"]; got != 2 {
		t.Errorf("after 2nd collect: expected PollCount=2, got %d", got)
	}
}

func TestCompress(t *testing.T) {
	input := []byte(`{"id":"Alloc","type":"gauge","value":123.45}`)

	compressed, err := compress(input)
	if err != nil {
		t.Errorf("compress error: %v", err)
	}
	t.Log(compressed)

	gz, err := gzip.NewReader(compressed)
	if err != nil {
		t.Errorf("gz read error: %v", err)
	}
	defer gz.Close()

	decompressed, err := io.ReadAll(gz)
	if err != nil {
		t.Errorf("decompress error: %v", err)
	}

	t.Log(string(input), string(decompressed))

	assert.Equal(t, input, decompressed)
}

func TestMemCollect_Batch(t *testing.T) {
	m := NewMemCollect(config.AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		SecretKey:      "",
		RateLimit:      30,
	})

	m.mu.Lock()
	for i := 0; i < 100; i++ {
		m.gauges[fmt.Sprintf("metric_%d", i)] = float64(i)
		log.Info().Msgf("metric_%d", i)
	}
	m.mu.Unlock()

	m.Add()

	select {
	case <-t.Context().Done():
		t.Errorf("Timeout waiting for batch: %v", t.Context().Err())
	case batch := <-m.jobs:
		if len(batch) != 100 {
			t.Errorf("invalid batch length, got %d, expected %d", len(batch), 100)
		}
	}
}
