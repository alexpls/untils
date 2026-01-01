package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgxlisten"
)

type DBEventHandler struct {
	s             *Service
	mu            sync.RWMutex
	subscriptions map[int64]map[chan struct{}]struct{}
}

func NewDBEventHandler(s *Service) *DBEventHandler {
	return &DBEventHandler{
		s:             s,
		subscriptions: make(map[int64]map[chan struct{}]struct{}),
	}
}

var _ pgxlisten.Handler = (*DBEventHandler)(nil)

func (d *DBEventHandler) HandleNotification(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
	monitorID, err := monitorIDFromPayload(notification.Payload)
	if err != nil {
		return fmt.Errorf("handling monitor_events notification: %w", err)
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	subscribers, ok := d.subscriptions[monitorID]
	if !ok {
		return nil
	}

	for ch := range subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	return nil
}

func (d *DBEventHandler) Subscribe(ctx context.Context, monitorID int64) <-chan struct{} {
	ch := make(chan struct{})

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.subscriptions[monitorID]; !ok {
		d.subscriptions[monitorID] = make(map[chan struct{}]struct{})
	}
	d.subscriptions[monitorID][ch] = struct{}{}

	go func() {
		<-ctx.Done()

		d.mu.Lock()
		defer d.mu.Unlock()

		delete(d.subscriptions[monitorID], ch)
		if len(d.subscriptions[monitorID]) == 0 {
			delete(d.subscriptions, monitorID)
		}

		close(ch)
	}()

	return ch
}

func monitorIDFromPayload(payload string) (int64, error) {
	var n struct {
		Table string `json:"table"`
	}

	if err := json.Unmarshal([]byte(payload), &n); err != nil {
		return 0, err
	}

	switch n.Table {
	case "monitors":
		var m struct {
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			return 0, fmt.Errorf("unmarshaling payload: %w", err)
		}
		return m.Data.ID, nil
	case "monitor_checks", "monitor_check_events", "monitor_results":
		var m struct {
			Data struct {
				MonitorID int64 `json:"monitor_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			return 0, fmt.Errorf("unmarshaling payload: %w", err)
		}
		return m.Data.MonitorID, nil
	default:
		return 0, fmt.Errorf("payload for unexpected table: %s", n.Table)
	}
}
