package chans

import "time"

// Debounce waits for a pause in events before sending, but guarantees delivery
// after maxWait even if events keep streaming in.
func Debounce[T any](source <-chan T, maxWait time.Duration) <-chan T {
	debounced := make(chan T)

	go func() {
		defer close(debounced)

		var (
			last     T
			have     bool
			debTimer *time.Timer // resets on each event
			maxTimer *time.Timer // fires after maxWait from first event
			debC     <-chan time.Time
			maxC     <-chan time.Time
		)

		send := func() {
			if have {
				debounced <- last
				have = false
			}
			if debTimer != nil {
				debTimer.Stop()
				debTimer = nil
				debC = nil
			}
			if maxTimer != nil {
				maxTimer.Stop()
				maxTimer = nil
				maxC = nil
			}
		}

		for {
			select {
			case msg, ok := <-source:
				if !ok {
					send()
					return
				}

				last = msg
				have = true

				if debTimer == nil {
					debTimer = time.NewTimer(maxWait)
					debC = debTimer.C
				} else {
					if !debTimer.Stop() {
						select {
						case <-debTimer.C:
						default:
						}
					}
					debTimer.Reset(maxWait)
				}

				if maxTimer == nil {
					maxTimer = time.NewTimer(maxWait)
					maxC = maxTimer.C
				}

			case <-debC:
				// no new events within window, send now
				send()

			case <-maxC:
				// max wait reached, send even if events are still coming
				send()
			}
		}
	}()

	return debounced
}
