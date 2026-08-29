package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler/mocks"
	"go.uber.org/mock/gomock"
)

func TestPingDatabase(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
		status  int
	}{
		{
			"Valid Connection",
			nil,
			http.StatusOK,
		},
		{
			"Invalid Connection",
			errors.New("failed connection"),
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPinger := mocks.NewMockPinger(ctrl)
			mockPinger.EXPECT().
				Ping(gomock.Any()).
				Return(tt.pingErr).
				Times(1)

			h := NewPingDB(mockPinger)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()
			h.PingDatabase(w, req)

			if w.Code != tt.status {
				t.Errorf("Expected status code %d, got %d", tt.status, w.Code)
			}
		})
	}
}

func TestNoopPinger_Ping(t *testing.T) {
	np := &NoopPinger{}
	err := np.Ping(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
