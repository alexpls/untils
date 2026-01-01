package db

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/wideevents"
	"github.com/jackc/pgx/v5"
)

type DBLogEvent struct {
	QueriesCount         int
	QueriesTotalDuration time.Duration
}

func (d *DBLogEvent) Key() string {
	return "db"
}

func (d *DBLogEvent) SlogAttr() slog.Attr {
	return slog.Group(d.Key(),
		slog.Int("query_count", d.QueriesCount),
		slog.Duration("query_total_duration", d.QueriesTotalDuration),
	)
}

var whitespaceCollapse = regexp.MustCompile(`\s+`)

type loggingTracer struct {
	logger *slog.Logger
}

func (t loggingTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	sql := whitespaceCollapse.ReplaceAllString(data.SQL, " ")

	if strings.Contains(sql, "river_") || sql == "begin" || sql == "commit" {
		// skip low signal queries
		return ctx
	}
	sql = strings.TrimSpace(sql)

	ctx = context.WithValue(ctx, "queryStartTime", time.Now())
	ctx = context.WithValue(ctx, "querySQL", sql)

	return ctx
}

func (t loggingTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	start, ok := ctx.Value("queryStartTime").(time.Time)
	if !ok {
		return
	}
	sql, ok := ctx.Value("querySQL").(string)
	if !ok {
		return
	}

	if ev, ok := wideevents.GetOrCreateFromContext(ctx, func() *DBLogEvent {
		return &DBLogEvent{}
	}); ok {
		ev.QueriesCount++
		ev.QueriesTotalDuration += time.Since(start)
	} else {
		t.logger.DebugContext(ctx, "SQL query", "q", sql, "duration_ms", time.Since(start).Milliseconds())
	}
}
