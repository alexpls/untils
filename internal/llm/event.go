package llm

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/wideevents"
)

// LLMEvent tracks statistics for an LLM workflow execution.
// It implements wideevents.Event for integration with wide event logging.
type LLMEvent struct {
	Start        time.Time
	End          time.Time
	Turns        []*LLMTurn
	SitesVisited []string
}

var _ wideevents.Event = &LLMEvent{}

// LLMTurn tracks statistics for a single LLM turn (request/response cycle).
type LLMTurn struct {
	Start     time.Time
	End       time.Time
	Cost      float64
	ToolCalls map[string]int
	Error     error
}

func newLLMEvent() *LLMEvent {
	return &LLMEvent{
		Start: time.Now(),
	}
}

func (e *LLMEvent) Key() string {
	return "llm"
}

func (e *LLMEvent) SlogAttr() slog.Attr {
	var totalCost float64
	totalToolCalls := make(map[string]int)
	success := true

	turnAttrs := make([]any, 0, len(e.Turns))
	for i, t := range e.Turns {
		totalCost += t.Cost
		for name, count := range t.ToolCalls {
			totalToolCalls[name] += count
		}
		if t.Error != nil {
			success = false
		}
		turnAttrs = append(turnAttrs, t.slogAttr(i+1))
	}

	var toolCallAttrs []any
	for name, count := range totalToolCalls {
		toolCallAttrs = append(toolCallAttrs, slog.Int(name, count))
	}

	return slog.Group(e.Key(),
		slog.Int("num_turns", len(e.Turns)),
		slog.Float64("total_cost_usd", totalCost),
		slog.Duration("total_duration", e.duration()),
		slog.String("sites_visited", strings.Join(e.SitesVisited, ", ")),
		slog.Bool("success", success),
		slog.Group("tool_calls", toolCallAttrs...),
		slog.Group("turn", turnAttrs...),
	)
}

func (e *LLMEvent) duration() time.Duration {
	if e.End.IsZero() {
		return time.Since(e.Start)
	}
	return e.End.Sub(e.Start)
}

func (e *LLMEvent) finish() {
	e.End = time.Now()
}

func (e *LLMEvent) newTurn() *LLMTurn {
	t := &LLMTurn{
		Start:     time.Now(),
		ToolCalls: make(map[string]int),
	}
	e.Turns = append(e.Turns, t)
	return t
}

func (e *LLMEvent) addSiteVisited(url string) {
	e.SitesVisited = append(e.SitesVisited, url)
}

func (t *LLMTurn) duration() time.Duration {
	if t.End.IsZero() {
		return time.Since(t.Start)
	}
	return t.End.Sub(t.Start)
}

func (t *LLMTurn) finish() {
	t.End = time.Now()
}

func (t *LLMTurn) addCost(cost float64) {
	t.Cost += cost
}

func (t *LLMTurn) addError(err error) {
	if err != nil {
		t.Error = err
	}
}

func (t *LLMTurn) incrToolCall(name string) {
	if t.ToolCalls == nil {
		t.ToolCalls = make(map[string]int)
	}
	t.ToolCalls[name]++
}

func (t *LLMTurn) slogAttr(num int) slog.Attr {
	var toolCallAttrs []any
	for name, count := range t.ToolCalls {
		toolCallAttrs = append(toolCallAttrs, slog.Int(name, count))
	}

	return slog.Group(strconv.Itoa(num),
		slog.Duration("duration", t.duration()),
		slog.Float64("cost_usd", t.Cost),
		slog.Bool("success", t.Error == nil),
		slog.Group("tool_calls", toolCallAttrs...),
	)
}
