#!make
include .env

POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
POSTGRES_DB ?= postgres

migrate-create:
	migrate create -ext sql -dir repository/postgres/migrations -seq $(name)

migrate-up:
	migrate -path repository/postgres/migrations -database "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" -verbose up

migrate-down:
	migrate -path repository/postgres/migrations -database "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" -verbose down
