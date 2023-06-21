#!make
include .env

POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
POSTGRES_DB ?= wordle

migrate-create:
	migrate create -ext sql -dir repository/postgres/migrations -seq $(name)

migrate-up:
	migrate -path repository/postgres/migrations -database "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" -verbose up

migrate-down:
	migrate -path repository/postgres/migrations -database "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" -verbose down

sqlc:
	sqlc generate -f ./sqlc.yaml

build:
	docker build -t dekuyo/wordle-server:$(TAG) -f ./Dockerfile --target production .
	docker image push dekuyo/wordle-server:$(TAG)

run: 
	rm -rf ./bin/main
	go build -o ./bin/main ./cmd/main.go
	./bin/main

deploy:
	docker stack rm wordle 2>>/dev/null
	docker stack deploy -c ./stack.yaml wordle

test: 
	go test ./... -json --cover | tparse -all