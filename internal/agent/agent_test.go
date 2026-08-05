package agent

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
)

func TestMemCollect(t *testing.T) {
	m := NewMemCollect(config.Config{
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
	m := NewMemCollect(config.Config{
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

func TestCounterCapitalize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)

	m := NewMemCollect(config.Config{
		Address:        u.Host,
		ReportInterval: 10,
		PollInterval:   2,
	})

	m.collect()
	m.collect()
	m.collect()

	if got := m.counters["PollCount"]; got != 3 {
		t.Errorf("expected PollCount=3, got %d", got)
	}
	t.Logf("got PollCount delta before send: %d", m.counters["PollCount"])

	m.Send()

	if got := m.counters["PollCount"]; got != 0 {
		t.Errorf("expected PollCount=0, got %d", got)
	}

	m.collect()
	m.collect()

	if got := m.counters["PollCount"]; got != 2 {
		t.Errorf("expected PollCount=2 after 2 more collects post-reset, got %d", got)
	}
	t.Logf("got PollCount delta in next iter: %d", m.counters["PollCount"])
}
