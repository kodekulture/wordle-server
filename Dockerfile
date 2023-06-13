FROM golang:1.20 AS development
WORKDIR /wordle
COPY . .
RUN go mod download
RUN go install github.com/cespare/reflex@latest
CMD reflex -sr '\.go$' go run ./cmd/main.go

FROM golang:alpine AS builder
COPY cmd cmd
COPY game game
COPY handler handler
COPY repository repository
COPY service service
COPY go.mod go.sum ./
COPY --from=development /usr/local/go /usr/local/go
RUN go build -o /go/bin/wordle-server ./cmd/main.go

FROM alpine:latest AS production
COPY --from=builder /go/bin/wordle-server /go/bin/wordle-server
COPY ./docs /docs
ENTRYPOINT ["/go/bin/wordle-server"]
