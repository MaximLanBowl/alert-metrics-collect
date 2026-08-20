package repository

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/models"
)

type Producer struct {
	file    *os.File
	encoder *json.Encoder
}

func NewProducer(filename string) (*Producer, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &Producer{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

func (p *Producer) WriteMetrics(metrics []models.Metrics) error {
	if err := p.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	if _, err := p.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	return p.encoder.Encode(metrics)
}

func (p *Producer) Close() error {
	return p.file.Close()
}

type Consumer struct {
	file    *os.File
	decoder *json.Decoder
}

func NewConsumer(filename string) (*Consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &Consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

func (c *Consumer) ReadMetrics() ([]models.Metrics, error) {
	var data []models.Metrics

	if err := c.decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode metrics: %w", err)
	}

	return data, nil
}

func (c *Consumer) Close() error {
	return c.file.Close()
}
