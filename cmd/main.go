package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/caarlos0/env/v8"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chat-Map/wordle-server/handler"
	"github.com/Chat-Map/wordle-server/handler/token"
	"github.com/Chat-Map/wordle-server/repository/postgres"
	"github.com/Chat-Map/wordle-server/service"
)

var port string

func main() {
	ctx := context.Background()
	err := env.Parse(&config)
	if err != nil {
		log.Fatal(err)
	}
	db, err := getConnection(ctx)
	if err != nil {
		log.Fatal(err)
	}
	srv := service.New(postgres.NewGameRepo(db), postgres.NewPlayerRepo(db))
	tokener, err := token.New([]byte(config.PASETOKey), "", time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	h := handler.New(srv, tokener)
	h.Start(port)
}

func getConnection(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := config.PostgresURL
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	conn, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	err = conn.Ping(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to ping database"))
	}
	return conn, nil
}

var config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	PostgresURL string `env:"POSTGRES_URL"`
	PASETOKey   string `env:"PASETO_KEY,required"`
}
