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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"

	"github.com/Chat-Map/wordle-server/handler"
	"github.com/Chat-Map/wordle-server/handler/token"
	"github.com/Chat-Map/wordle-server/repository/postgres"
	"github.com/Chat-Map/wordle-server/service"
)

func readInConfig() error {
	viper.SetConfigFile(".env") // read from .env
	viper.AutomaticEnv()        // read from env
	if err := viper.ReadInConfig(); err != nil {
		return errors.Join(err, errors.New("failed to read in config"))
	}
	return viper.Unmarshal(&config)
}

func main() {
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := readInConfig()
	if err != nil {
		log.Fatal(err)
	}
	db, err := getConnection(appCtx)
	if err != nil {
		log.Fatal(err)
	}
	srv := service.New(appCtx, postgres.NewGameRepo(db), postgres.NewPlayerRepo(db))
	tokener, err := token.New([]byte(config.PASETOKey), "", time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	h := handler.New(srv, tokener)
	go shutdown(h)
	log.Printf("server started on port: %s", config.Port)
	if err = h.Start(config.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func getConnection(ctx context.Context) (*pgxpool.Pool, error) {
	conn, err := pgxpool.New(ctx, config.PostgresURL)
	if err != nil {
		return nil, err
	}
	err = conn.Ping(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to ping database"))
	}
	return conn, nil
}

func shutdown(s *handler.Handler) {
	// Wait for interrupt signal to gracefully shutdown the server with
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := s.Stop(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

var config struct {
	Port        string `mapstructure:"PORT"`
	PostgresURL string `mapstructure:"POSTGRES_URL"`
	PASETOKey   string `mapstructure:"PASETO_KEY"`
}
