package agent

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/rs/zerolog/log"
)

type MemCollect struct {
	mu             sync.Mutex
	gauges         map[string]float64
	counters       map[string]int64
	baseURL        string
	client         *http.Client
	pollInterval   time.Duration
	reportInterval time.Duration
}

func NewMemCollect(cfg config.Config) *MemCollect {
	return &MemCollect{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		baseURL:  "http://" + cfg.Address,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
	}
}

func (m *MemCollect) collect() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges["Alloc"] = float64(stats.Alloc)
	m.gauges["BuckHashSys"] = float64(stats.BuckHashSys)
	m.gauges["Frees"] = float64(stats.Frees)
	m.gauges["GCCPUFraction"] = float64(stats.GCCPUFraction)
	m.gauges["GCSys"] = float64(stats.GCSys)
	m.gauges["HeapAlloc"] = float64(stats.HeapAlloc)
	m.gauges["HeapIdle"] = float64(stats.HeapIdle)
	m.gauges["HeapInuse"] = float64(stats.HeapInuse)
	m.gauges["HeapObjects"] = float64(stats.HeapObjects)
	m.gauges["HeapReleased"] = float64(stats.HeapReleased)
	m.gauges["HeapSys"] = float64(stats.HeapSys)
	m.gauges["LastGC"] = float64(stats.LastGC)
	m.gauges["Lookups"] = float64(stats.Lookups)
	m.gauges["MCacheInuse"] = float64(stats.MCacheInuse)
	m.gauges["MCacheSys"] = float64(stats.MCacheSys)
	m.gauges["MSpanInuse"] = float64(stats.MSpanInuse)
	m.gauges["MSpanSys"] = float64(stats.MSpanSys)
	m.gauges["Mallocs"] = float64(stats.Mallocs)
	m.gauges["NextGC"] = float64(stats.NextGC)
	m.gauges["NumForcedGC"] = float64(stats.NumForcedGC)
	m.gauges["NumGC"] = float64(stats.NumGC)
	m.gauges["OtherSys"] = float64(stats.OtherSys)
	m.gauges["PauseTotalNs"] = float64(stats.PauseTotalNs)
	m.gauges["StackInuse"] = float64(stats.StackInuse)
	m.gauges["StackSys"] = float64(stats.StackSys)
	m.gauges["Sys"] = float64(stats.Sys)
	m.gauges["TotalAlloc"] = float64(stats.TotalAlloc)
	m.gauges["RandomValue"] = rand.Float64()

	m.counters["PollCount"]++
}

func (m *MemCollect) sendGauge(name string, value float64) error {
	url := fmt.Sprintf(m.baseURL+"/update/gauge/%s/%s", name, strconv.FormatFloat(value, 'f', -1, 64))
	return m.post(url)
}

func (m *MemCollect) sendCounter(name string, delta int64) error {
	url := fmt.Sprintf(m.baseURL+"/update/counter/%s/%d", name, delta)
	return m.post(url)
}

func (m *MemCollect) post(url string) error {
	response, err := m.client.Post(url, "text/plain", nil)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	return nil
}

func (m *MemCollect) Send() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, value := range m.gauges {
		if err := m.sendGauge(name, value); err != nil {
			log.Error().Err(err).Msgf("failed to send gauge %s", name)
			continue
		}
	}

	for name, delta := range m.counters {
		if err := m.sendCounter(name, delta); err != nil {
			log.Error().Err(err).Msgf("failed to send counter %s", name)
			continue
		}

		m.counters[name] = 0
	}
}

func (m *MemCollect) Run() {
	go func() {
		for {
			time.Sleep(m.pollInterval)
			m.collect()
			log.Info().Msg("Runtime metrics collected")
		}
	}()

	for {
		time.Sleep(m.reportInterval)
		m.Send()
		log.Info().Msg("Metrics sent")
	}
}
