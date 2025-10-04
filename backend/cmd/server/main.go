package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"simstack/internal/server"
)

func main() {
	addr := getEnv("SIMSTACK_ADDR", ":8080")

	srv := server.NewServer()

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("SimStack backend listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
