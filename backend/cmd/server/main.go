// Command server is the CommitIn backend HTTP API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Platon223/commitin/backend/internal/config"
	"github.com/Platon223/commitin/backend/internal/db"
	"github.com/Platon223/commitin/backend/internal/user"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	client, database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("shutdown: mongo disconnect: %v", err)
		}
	}()

	users := user.NewStore(database)
	if err := users.EnsureIndexes(ctx); err != nil {
		log.Fatalf("startup: ensure user indexes: %v", err)
	}
	log.Printf("connected to MongoDB %q, user indexes ensured", cfg.MongoDB)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: http: %v", err)
	}
	log.Print("shutdown complete")
}
