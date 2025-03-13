package main

import (
	"log"

	"github.com/Zireael26/broken-links-checker/internal/server"
)

func main() {
    srv, err := server.New()
    if err != nil {
        log.Fatalf("Failed to initialize server: %v", err)
    }

    log.Printf("Starting BLC API on :8080")
    if err := srv.Start(); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}