package llm

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type turn struct {
	start     time.Time
	end       time.Time
	err       error
	cost      float64        // USD cents
	toolCalls map[string]int // maps tool name to number of times invoked
}

func (t turn) duration() time.Duration {
	return t.end.Sub(t.start)
}

func (t *turn) incrToolCall(name string) {
	_, ok := t.toolCalls[name]
	if !ok {
		t.toolCalls[name] = 0
	}
	t.toolCalls[name] += 1
}

type stats struct {
	turns []*turn
}

func (s *stats) newTurn() *turn {
	t := &turn{start: time.Now(), toolCalls: make(map[string]int)}
	s.turns = append(s.turns, t)
	return t
}

func (s stats) totalCost() float64 {
	var c float64
	for _, t := range s.turns {
		c += t.cost
	}
	return c
}

func (s stats) totalToolCalls() map[string]int {
	calls := make(map[string]int)
	for _, t := range s.turns {
		for name, count := range t.toolCalls {
			_, ok := calls[name]
			if !ok {
				calls[name] = 0
			}
			calls[name] += count
		}
	}
	return calls
}

func (s stats) totalDuration() time.Duration {
	if len(s.turns) == 0 {
		return 0
	}
	first := s.turns[0]
	last := s.turns[len(s.turns)-1]
	return last.end.Sub(first.start)
}

func (s stats) errors() error {
	var errs []error
	for _, turn := range s.turns {
		if turn.err != nil {
			errs = append(errs, turn.err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *stats) log(logger *slog.Logger) {
	var toolCallAttrs []any
	for name, count := range s.totalToolCalls() {
		toolCallAttrs = append(toolCallAttrs, slog.Int(name, count))
	}

	var outcomeAttrs []any
	errs := s.errors()
	outcomeAttrs = append(outcomeAttrs, slog.Bool("success", errs == nil))
	if errs != nil {
		outcomeAttrs = append(outcomeAttrs, slog.String("error", errs.Error()))
	}

	logger.Info("llm stats",
		slog.Int("num_turns", len(s.turns)),
		slog.Float64("total_cost_usd", s.totalCost()),
		slog.Duration("total_duration", s.totalDuration()),
		slog.Group("tool_calls", toolCallAttrs...),
	)
}

var contextKey = struct{}{}

func statsFromContext(ctx context.Context) *stats {
	val, ok := ctx.Value(contextKey).(*stats)
	if !ok {
		panic("didn't find stats on context")
	}
	return val
}

func withStatsContext(ctx context.Context) (context.Context, *stats) {
	if existing, ok := ctx.Value(contextKey).(*stats); ok {
		return ctx, existing
	}
	s := &stats{}
	return context.WithValue(ctx, contextKey, s), s
}
