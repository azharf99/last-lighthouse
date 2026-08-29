package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lastlighthouse/lastlighthouse/core"
	"github.com/lastlighthouse/lastlighthouse/internal/auth"
	"github.com/lastlighthouse/lastlighthouse/internal/match"
	"github.com/lastlighthouse/lastlighthouse/internal/store"
	transporthttp "github.com/lastlighthouse/lastlighthouse/internal/transport/http"
	"github.com/lastlighthouse/lastlighthouse/internal/transport/ws"
)

func main() {
	addr := flag.String("addr", getEnv("LLH_ADDR", ":8080"), "HTTP/WebSocket listen address")
	dbURL := flag.String("db", getEnv("LLH_DB_URL", ""), "PostgreSQL connection URL (empty = memory store)")
	jwtSecret := flag.String("jwt-secret", getEnv("LLH_JWT_SECRET", ""), "JWT secret for guest auth")
	turnTimeoutStr := flag.String("turn-timeout", getEnv("LLH_TURN_TIMEOUT", "90s"), "Turn timeout duration")
	flag.Parse()

	turnTimeout, err := time.ParseDuration(*turnTimeoutStr)
	if err != nil {
		log.Fatalf("invalid turn-timeout %q: %v", *turnTimeoutStr, err)
	}

	content := core.DefaultContent()
	log.Printf("The Last Lighthouse — Server Online (M2)")
	log.Printf("Konten hash: %s", content.Hash)

	// 1. Inisialisasi Store (Postgres atau Memory)
	var s store.Store
	if *dbURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pgStore, err := store.NewPostgresStore(ctx, *dbURL)
		if err != nil {
			log.Fatalf("gagal inisialisasi postgres store: %v", err)
		}
		s = pgStore
		log.Printf("Store: PostgreSQL terhubung (%s)", maskDBURL(*dbURL))
	} else {
		s = store.NewMemoryStore()
		log.Printf("Store: In-Memory (development mode, tanpa persistensi disk)")
	}
	defer s.Close()

	// 2. Inisialisasi Auth
	var secretBytes []byte
	if *jwtSecret != "" {
		secretBytes = []byte(*jwtSecret)
	}
	authenticator := auth.NewAuthenticator(secretBytes)

	// 3. Inisialisasi Match Registry & Actor Engine
	registry := match.NewRegistry(s, content, turnTimeout)
	defer registry.Shutdown()

	// 4. Inisialisasi WebSocket Hub & HTTP Server
	hub := ws.NewHub(registry, s, authenticator)
	httpServer := transporthttp.NewServer(authenticator, s, registry, hub)

	server := &http.Server{
		Addr:         *addr,
		Handler:      httpServer.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// 5. Jalankan server HTTP & Deadline Scheduler (M5/M6)
	if httpServer.Scheduler() != nil {
		httpServer.Scheduler().Start()
		defer httpServer.Scheduler().Stop()
	}

	go func() {
		log.Printf("Server mendengarkan di %s", *addr)
		log.Printf("WebSocket endpoint: ws://%s/ws", *addr)
		log.Printf("Lobby API: http://%s/api/lobby", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 6. Graceful shutdown on SIGINT / SIGTERM (ADR-004)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Sinyal shutdown diterima. Menyimpan snapshot match aktif...")
	if httpServer.Scheduler() != nil {
		httpServer.Scheduler().Stop()
	}
	registry.Shutdown()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	log.Println("Server berhasil dihentikan dengan aman.")
}

func getEnv(key, defVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defVal
}

func maskDBURL(url string) string {
	if len(url) <= 12 {
		return "***"
	}
	return fmt.Sprintf("%s...%s", url[:8], url[len(url)-4:])
}
