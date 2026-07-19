package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Kartik-2239/pinwheel/internal/auth"
	"github.com/Kartik-2239/pinwheel/internal/db"
	"github.com/Kartik-2239/pinwheel/internal/proxy"
)

func main() {
	dbPath := os.Getenv("PROXY_DB_PATH")
	if dbPath == "" {
		dbPath = "proxy.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	store := db.NewStore(database)

	p := proxy.New(store)
	middleware := auth.Middleware(store)
	handler := middleware(p)

	mux := http.NewServeMux()

	mux.Handle("/", handler)

	server := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("listening on :8081")
	log.Fatal(server.ListenAndServe())
}
