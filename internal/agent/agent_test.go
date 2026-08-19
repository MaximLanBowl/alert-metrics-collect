package agent

import (
	"compress/gzip"
	"io"
	"testing"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMemCollect(t *testing.T) {
	m := NewMemCollect(config.AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
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

func TestPollCountIncrements(t *testing.T) {
	m := NewMemCollect(config.AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
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
