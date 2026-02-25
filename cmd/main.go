package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	switch subcommand() {
	case "serve":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		globalCfg, c := parseServe()
		a, appCtx, cancelAppCtx, appCloser := createApp(globalCfg)
		defer appCloser()

		addr := fmt.Sprintf(":%d", c.port)
		srv := &http.Server{
			Addr:    addr,
			Handler: a.routes(),
			BaseContext: func(l net.Listener) context.Context {
				return appCtx
			},
		}
		srvErrs := make(chan error, 1)

		go func() {
			a.logger.InfoContext(appCtx, "starting http server", "port", c.port)
			srvErrs <- srv.ListenAndServe()
		}()

		select {
		case err := <-srvErrs:
			a.logger.ErrorContext(appCtx, "http server error", "error", err)
			return
		case <-ctx.Done():
			a.logger.InfoContext(appCtx, "http server received shutdown signal")

			// Cancel app context first so long-lived handlers (e.g. SSE streams) exit quickly.
			cancelAppCtx()

			tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := srv.Shutdown(tctx); err != nil {
				a.logger.ErrorContext(tctx, "graceful http shutdown failed", "error", err)
				if err := srv.Close(); err != nil {
					a.logger.ErrorContext(tctx, "forcing http server close failed", "error", err)
				}
			}

			a.logger.InfoContext(tctx, "http server stopped")
		}
	case "seed":
		globalCfg := parseSeed()

		a, _, _, appCloser := createApp(globalCfg)
		defer appCloser()

		a.seed()

	case "migrate":
		c := parseMigrate()

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		logger.Info("starting database migration")

		if err := runMigrations(logger, c.dbUrl); err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}

	default:
		panic("unhandled subcommand")
	}
}
