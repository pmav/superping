package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/superping/internal/monitor"
	"github.com/example/superping/internal/ping"
	"github.com/example/superping/internal/storage"
	_ "modernc.org/sqlite"
)

//go:embed web
var webFiles embed.FS

func main() {
	databasePath := envOrDefault("SUPERPING_DATABASE", "superping.db")
	address := envOrDefault("SUPERPING_ADDRESS", ":8080")

	database, err := sql.Open("sqlite", sqliteDSN(databasePath))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	store := storage.NewStore(database)
	if err := store.Initialize(context.Background()); err != nil {
		log.Fatalf("initialize database: %v", err)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           newServerHandler(store),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	monitorContext, stopMonitor := context.WithCancel(context.Background())
	defer stopMonitor()
	go monitor.New(store).Run(monitorContext)
	go runScheduledTask(monitorContext, store)

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", address)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	case <-shutdownSignal:
		stopMonitor()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}
}

func newServerHandler(store ping.Store) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/pings", ping.NewHandler(store))
	mux.Handle("/pings/", ping.NewHandler(store))
	webRoot, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(webRoot)))
	return mux
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func sqliteDSN(databasePath string) string {
	separator := "?"
	if strings.Contains(databasePath, "?") {
		separator = "&"
	}
	return databasePath + separator + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}
