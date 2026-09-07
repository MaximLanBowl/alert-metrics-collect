package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/models"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/wrappers"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type MemCollect struct {
	mu             sync.Mutex
	wg             sync.WaitGroup
	jobs           chan []models.Metrics
	gauges         map[string]float64
	counters       map[string]int64
	baseURL        string
	client         *http.Client
	pollInterval   time.Duration
	reportInterval time.Duration
	secretKey      string
	rateLimit      int
}

func NewMemCollect(cfg config.AgentConfig) *MemCollect {
	return &MemCollect{
		jobs:     make(chan []models.Metrics, cfg.RateLimit),
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		baseURL:  "http://" + cfg.Address,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
		secretKey:      cfg.SecretKey,
		rateLimit:      cfg.RateLimit,
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

func (m *MemCollect) collectMemCPU() {
	memory, err := mem.VirtualMemory()
	if err != nil {
		log.Error().Err(err).Msg("failed to get memory stats")
		return
	}

	percentCPU, err := cpu.Percent(0, true)
	if err != nil {
		log.Error().Err(err).Msg("failed to get cpu utilization")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges["TotalMemory"] = float64(memory.Total)
	m.gauges["FreeMemory"] = float64(memory.Free)

	if len(percentCPU) > 0 {
		for i, p := range percentCPU {
			name := fmt.Sprintf("CPUutilization%d", i+1)
			m.gauges[name] = p
		}
	}
}

func (m *MemCollect) sendGauge(name string, value float64) error {
	metric := models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}

	return m.post(metric)
}

func (m *MemCollect) sendCounter(name string, delta int64) error {
	metric := models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	}

	return m.post(metric)
}

func compress(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	wr := gzip.NewWriter(&buf)
	if _, err := wr.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to compressor: %w", err)
	}

	err := wr.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close compressor: %w", err)
	}

	return &buf, nil
}

func (m *MemCollect) post(metric models.Metrics) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	cmpr, err := compress(body)
	if err != nil {
		return fmt.Errorf("failed to compress request body: %w", err)
	}

	cmprBytes := cmpr.Bytes()

	err = wrappers.WithRetry(func() error {
		req, err := http.NewRequest(http.MethodPost, m.baseURL+"/update", bytes.NewReader(cmprBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		log.Info().RawJSON("request", body).Msg("Request body")

		resp, err := m.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to send request: %s", resp.Status)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
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

func (m *MemCollect) worker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.collect()
				log.Info().Msg("Runtime metrics collected")
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(m.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.collectMemCPU()
				log.Info().Msg("Runtime CPU metrics collected")
			}
		}
	}()
}

func (m *MemCollect) Run(ctx context.Context) {
	for i := 0; i < m.rateLimit; i++ {
		m.wg.Add(1)
		go m.sendBatch()
	}

	go m.worker(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(m.reportInterval):
			m.Add()
			log.Info().Msg("Metrics add")
		}
	}
}

func (m *MemCollect) sendBatch() {
	defer m.wg.Done()
	for bt := range m.jobs {
		if err := m.flush(bt); err != nil {
			log.Error().Err(err).Msg("failed to send batch")
		}
	}
}

func (m *MemCollect) flush(metrics []models.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	hashStr := m.calcHash(body, m.secretKey)

	cmpr, err := compress(body)
	if err != nil {
		return fmt.Errorf("failed to compress metrics: %w", err)
	}
	cmprBytes := cmpr.Bytes()

	err = wrappers.WithRetry(func() error {
		req, err := http.NewRequest(http.MethodPost, m.baseURL+"/updates/", bytes.NewReader(cmprBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if m.secretKey != "" {
			req.Header.Set("HashSHA256", hashStr)
		}
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		log.Info().Msgf("url in request: %s", req.URL.String())
		log.Info().RawJSON("request", body).Msg("Request body")
		resp, err := m.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to send request: %s", resp.Status)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}

func (m *MemCollect) Add() {
	m.mu.Lock()
	defer m.mu.Unlock()

	batch := make([]models.Metrics, 0, len(m.gauges)+len(m.counters))

	for name, value := range m.gauges {
		batch = append(batch, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		})
	}

	for name, delta := range m.counters {
		batch = append(batch, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &delta,
		})
		m.counters[name] = 0
	}

	if len(batch) == 0 {
		log.Debug().Msg("no metrics to send")
		return
	}

	m.jobs <- batch
}

func (m *MemCollect) Close() {
	close(m.jobs)
	m.wg.Wait()
}

func (m *MemCollect) calcHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
