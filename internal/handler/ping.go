package handler

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
)

//go:generate mockgen -source=ping.go -destination=mocks/ping_mock.go -package=mocks

type Pinger interface {
	Ping(ctx context.Context) error
}

type NoopPinger struct{}

func (np *NoopPinger) Ping(_ context.Context) error {
	return nil
}

type PingDBHandler struct {
	connPool Pinger
}

func NewPingDB(connPool Pinger) *PingDBHandler {
	return &PingDBHandler{
		connPool: connPool,
	}
}

func (p *PingDBHandler) PingDatabase(w http.ResponseWriter, r *http.Request) {
	if err := p.connPool.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to ping database")
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Info().Msg("Ping successful")
}
