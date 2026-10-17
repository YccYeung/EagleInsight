// Command eagleinsight serves the article API and, when FRONTEND_DIR is set,
// the PWA frontend alongside it.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"eagleinsight/internal/api"
	"eagleinsight/internal/store"
)

const refreshInterval = 3 * time.Hour

func main() {
	dbPath := getenv("DB_PATH", "eagleinsight.db")
	port := getenv("PORT", "8080")

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	srv := api.NewServer(st)

	if count, _ := st.ArticleCount(); count == 0 {
		log.Println("seeding articles on first run…")
		if n, err := srv.Refresh(context.Background()); err != nil {
			log.Printf("initial refresh failed: %v", err)
		} else {
			log.Printf("seeded %d articles", n)
		}
	}

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			n, err := srv.Refresh(ctx)
			cancel()
			if err != nil {
				log.Printf("scheduled refresh failed: %v", err)
			} else {
				log.Printf("scheduled refresh: %d new articles", n)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Routes())
	if frontendDir := os.Getenv("FRONTEND_DIR"); frontendDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(frontendDir)))
	}

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
