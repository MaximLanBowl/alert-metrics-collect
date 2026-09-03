package agent

import (
	"bytes"
	"compress/gzip"
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
	models "github.com/MaximLanBowl/alert-metrics-collect/internal/models"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/wrappers"
	"github.com/rs/zerolog/log"
)

type MemCollect struct {
	mu             sync.Mutex
	wg             sync.WaitGroup
	mtBatch        chan []models.Metrics
	gauges         map[string]float64
	counters       map[string]int64
	baseURL        string
	client         *http.Client
	pollInterval   time.Duration
	reportInterval time.Duration
	secretKey      string
}

func NewMemCollect(cfg config.AgentConfig) *MemCollect {
	return &MemCollect{
		mtBatch:  make(chan []models.Metrics, 1),
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		baseURL:  "http://" + cfg.Address,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		reportInterval: time.Duration(cfg.ReportInterval) * time.Second,
		pollInterval:   time.Duration(cfg.PollInterval) * time.Second,
		secretKey:      cfg.SecretKey,
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

func (m *MemCollect) Run() {
	m.wg.Add(1)
	go m.sendBatch()

	go func() {
		for {
			time.Sleep(m.pollInterval)
			m.collect()
			log.Info().Msg("Runtime metrics collected")
		}
	}()

	for {
		time.Sleep(m.reportInterval)
		m.Add()
		log.Info().Msg("Metrics add")
	}
}

func (m *MemCollect) sendBatch() {
	defer m.wg.Done()
	for bt := range m.mtBatch {
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

	m.mtBatch <- batch
}

func (m *MemCollect) Close() {
	close(m.mtBatch)
	m.wg.Wait()
}

func (m *MemCollect) calcHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
