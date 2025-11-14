package main

import (
	"flag"
	"log"

	"github.com/krakovia/blockchain/pkg/signaling"
)

func main() {
	addr := flag.String("addr", ":9000", "Signaling server address")
	flag.Parse()

	server := signaling.NewServer()

	log.Printf("Starting signaling server on %s", *addr)
	if err := server.Start(*addr); err != nil {
		log.Fatal("Error starting signaling server:", err)
	}
}
