FROM golang:1.23 AS development
WORKDIR /wordle
COPY . .
RUN go mod download
RUN go install github.com/cespare/reflex@latest
CMD reflex -sr '\.go$' go run ./cmd/main.go

FROM golang:alpine AS builder
WORKDIR /wordle
COPY . .
RUN go build -o /go/bin/wordle-server ./cmd/main.go

FROM alpine:latest AS production
COPY --from=builder /go/bin/wordle-server /go/bin/wordle-server
ENTRYPOINT ["/go/bin/wordle-server"]
