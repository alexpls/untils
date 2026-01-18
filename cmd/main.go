package main

import (
	"context"
	"fmt"
	"log/slog"
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
		a, appCloser := createApp(globalCfg)
		defer appCloser()

		addr := fmt.Sprintf(":%d", c.port)
		srv := &http.Server{
			Addr:    addr,
			Handler: a.routes(),
		}
		srvErrs := make(chan error, 1)

		go func() {
			a.logger.Info("starting http server", "port", c.port)
			srvErrs <- srv.ListenAndServe()
		}()

		select {
		case err := <-srvErrs:
			a.logger.Error("http server error", "error", err)
			return
		case <-ctx.Done():
			a.logger.Info("http server received shutdown signal")

			tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := srv.Shutdown(tctx); err != nil {
				a.logger.Error("graceful http shutdown failed", "error", err)
				if err := srv.Close(); err != nil {
					a.logger.Error("forcing http server close failed", "error", err)
				}
			}

			a.logger.Info("http server stopped")
		}
	case "seed":
		globalCfg := parseSeed()

		a, appCloser := createApp(globalCfg)
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
