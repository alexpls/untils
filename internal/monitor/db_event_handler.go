package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgxlisten"
)

const throttleWindow = 100 * time.Millisecond

type subsMap map[chan struct{}]struct{}
type idToSubsMap map[int64]subsMap

type DBEventHandler struct {
	s             *Service
	mu            sync.RWMutex
	monitorIDSubs idToSubsMap
	userIDSubs    idToSubsMap
	subsRebuilt   time.Time
}

func NewDBEventHandler(s *Service) *DBEventHandler {
	return &DBEventHandler{
		s:             s,
		monitorIDSubs: make(idToSubsMap),
		userIDSubs:    make(idToSubsMap),
		subsRebuilt:   time.Now(),
	}
}

var _ pgxlisten.Handler = (*DBEventHandler)(nil)

type monitorEventPayload struct {
	Table     string `json:"table"`
	Action    string `json:"action"`
	MonitorID int64  `json:"monitor_id"`
	UserID    int64  `json:"user_id"`
}

func (d *DBEventHandler) HandleNotification(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
	var payload monitorEventPayload
	if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
		return fmt.Errorf("unmarshaling monitor_events notification: %w", err)
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	if subscribers, ok := d.monitorIDSubs[payload.MonitorID]; ok {
		for ch := range subscribers {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}

	if subscribers, ok := d.userIDSubs[payload.UserID]; ok {
		for ch := range subscribers {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}

	return nil
}

func (d *DBEventHandler) SubscribeMonitor(ctx context.Context, monitorID int64) <-chan struct{} {
	return throttle(d.subscribe(ctx, monitorID, d.monitorIDSubs), throttleWindow)
}

func (d *DBEventHandler) SubscribeUser(ctx context.Context, userID int64) <-chan struct{} {
	return throttle(d.subscribe(ctx, userID, d.userIDSubs), throttleWindow)
}

func (d *DBEventHandler) subscribe(ctx context.Context, id int64, subs idToSubsMap) <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()

	if time.Since(d.subsRebuilt) > time.Hour {
		d.rebuildSubsLocked()
	}

	ch := make(chan struct{})

	if _, ok := subs[id]; !ok {
		subs[id] = make(subsMap)
	}
	subs[id][ch] = struct{}{}

	go func() {
		<-ctx.Done()

		d.mu.Lock()
		defer d.mu.Unlock()

		delete(subs[id], ch)
		if len(subs[id]) == 0 {
			delete(subs, id)
		}

		close(ch)
	}()

	return ch
}

// rebuildSubsLocked creates new subscription maps to prevent memory leaks.
// This function must be called with the mutex already locked.
func (d *DBEventHandler) rebuildSubsLocked() {
	d.monitorIDSubs = rebuildSubMap(d.monitorIDSubs)
	d.userIDSubs = rebuildSubMap(d.userIDSubs)
	d.subsRebuilt = time.Now()
}

func rebuildSubMap(old idToSubsMap) idToSubsMap {
	newSubs := make(idToSubsMap)
	for id, chans := range old {
		newSubs[id] = make(subsMap)
		for ch := range chans {
			newSubs[id][ch] = struct{}{}
		}
	}
	return newSubs
}

func throttle[T any](source <-chan T, timeout time.Duration) <-chan T {
	throttled := make(chan T)

	go func() {
		defer close(throttled)

		var (
			last   T
			have   bool
			timer  *time.Timer
			timerC <-chan time.Time
		)

		for {
			select {
			case msg, ok := <-source:
				if !ok {
					if have {
						throttled <- last
					}
					return
				}

				last = msg
				have = true

				if timer == nil {
					timer = time.NewTimer(timeout)
					timerC = timer.C
				}

			case <-timerC:
				if have {
					throttled <- last
					have = false
				}

				timer.Stop()
				timer = nil
				timerC = nil
			}
		}
	}()

	return throttled
}
