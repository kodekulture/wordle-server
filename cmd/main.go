package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kodekulture/wordle-server/internal/config"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/kodekulture/wordle-server/handler"
	"github.com/kodekulture/wordle-server/handler/token"
	"github.com/kodekulture/wordle-server/repository/badgr"
	"github.com/kodekulture/wordle-server/repository/postgres"
	"github.com/kodekulture/wordle-server/service"
)

func main() {
	done := make(chan struct{})

	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	lvl, err := zerolog.ParseLevel(config.GetOrDefault("LOG_LEVEL", "debug", func(v string) (string, error) { return v, nil }))
	if err == nil {
		zerolog.SetGlobalLevel(lvl)
		zlog.WithLevel(lvl).Msgf("Setting log level to %v", lvl)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := getConnection(appCtx)
	if err != nil {
		log.Fatal(err)
	}
	cache, err := getCacher()
	if err != nil {
		log.Fatal(err)
	}
	srv, err := service.New(appCtx, postgres.NewGameRepo(db), postgres.NewPlayerRepo(db), badgr.New(cache))
	if err != nil {
		log.Fatal(err)
	}
	tokener, err := token.New([]byte(config.Get("PASETO_KEY")), "")
	if err != nil {
		log.Fatal(err)
	}
	h := handler.New(srv, tokener)
	go shutdown(h, done)
	log.Printf("server started on port: %s", config.Get("PORT"))
	if err = h.Start(config.Get("PORT")); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-done
}

func getConnection(ctx context.Context) (*pgxpool.Pool, error) {
	conn, err := pgxpool.New(ctx, config.Get("POSTGRES_URL"))
	if err != nil {
		return nil, err
	}
	err = conn.Ping(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to ping database"))
	}
	return conn, nil
}

func getCacher() (*badger.DB, error) {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	db, err := badger.Open(badger.DefaultOptions(config.Get("BADGER_PATH")))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func shutdown(s *handler.Handler, done chan<- struct{}) {
	// Wait for interrupt signal to gracefully shutdown the server with
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	log.Println("shutdown started")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := s.Stop(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("shutdown complete")
	close(done)
}
