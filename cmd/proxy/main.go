package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Kartik-2239/openai-proxy/internal/auth"
	"github.com/Kartik-2239/openai-proxy/internal/db"
	"github.com/Kartik-2239/openai-proxy/internal/proxy"
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

	p := proxy.New("http://127.0.0.1:8000")
	middleware := auth.Middleware(store)
	handler := middleware(p)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
