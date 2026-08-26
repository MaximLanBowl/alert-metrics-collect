package wrappers

import (
	"errors"
	"net"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

const (
	maxAttempts = 3
)

var retryIntervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

func isConnectionExceptionPG(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ConnectionException
}

func isConnectionExceptionAG(err error) bool {
	var netErr *net.OpError
	return errors.As(err, &netErr)
}

func WithRetry(fn func() error) error {
	var err error

	for i := 0; i <= maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if !isConnectionExceptionPG(err) && !isConnectionExceptionAG(err) {
			return err
		}

		if i == maxAttempts {
			log.Info().Msg("Failed to execute function after all retries")
			break
		}

		time.Sleep(retryIntervals[i])
	}

	return err
}
