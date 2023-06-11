package main

import (
	"flag"

	"github.com/Chat-Map/wordle-server/handler"
)

var port string

func main() {
	flag.StringVar(&port, port, "8080", "application port")
	flag.Parse()

	h := handler.New()
	h.Start(port)
}
