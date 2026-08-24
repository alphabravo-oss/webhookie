package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/config"
	"github.com/alphabravo-oss/webhookie/internal/httpapi"
	"github.com/alphabravo-oss/webhookie/internal/observe"
	"github.com/alphabravo-oss/webhookie/internal/sqlite"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.FromEnv()
	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	st := store.New(db.DB)
	srvAPI := httpapi.New(cfg, st, func(ctx context.Context) error { return st.Ping(ctx) })
	if err := srvAPI.Seed(context.Background()); err != nil {
		log.Error("seed sinks", "err", err)
		os.Exit(1)
	}

	httpSrv := &http.Server{Addr: cfg.Addr, Handler: srvAPI.Router()}
	go pruneLoop(st, cfg, log)

	go func() {
		log.Info("listening", "addr", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func pruneLoop(st *store.Store, cfg config.Config, log *slog.Logger) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		n, err := st.Prune(context.Background(), time.Duration(cfg.RetentionDays)*24*time.Hour, cfg.MaxEvents)
		if err != nil {
			log.Error("prune", "err", err)
			continue
		}
		if n > 0 {
			observe.Pruned.Add(int64(n))
		}
		if c, err := st.CountEvents(context.Background()); err == nil {
			observe.Stored.Store(int64(c))
		}
	}
}
