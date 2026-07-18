package agent

import "testing"

func TestMemCollect(t *testing.T) {
	m := NewMemCollect("http://localhost:8080")

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
	m := NewMemCollect("http://localhost:8080")

	m.collect()
	if got := m.counters["PollCount"]; got != 1 {
		t.Errorf("after 1st collect: expected PollCount=1, got %d", got)
	}

	m.collect()
	if got := m.counters["PollCount"]; got != 2 {
		t.Errorf("after 2nd collect: expected PollCount=2, got %d", got)
	}
}
