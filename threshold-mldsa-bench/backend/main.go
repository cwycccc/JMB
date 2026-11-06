package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"threshold-mldsa/backend/api"
	transactionstore "threshold-mldsa/backend/store"
)

func main() {
	address := os.Getenv("JMB_BACKEND_ADDR")
	if address == "" {
		address = ":8080"
	}

	server := &http.Server{
		Addr:              address,
		Handler:           api.NewHandler(transactionstore.New()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("clearing backend listening on %s", address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
