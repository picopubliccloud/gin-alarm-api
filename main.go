package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	db "github.com/picopubliccloud/alarm-api/internal/data/actual_data"
	ticketdb "github.com/picopubliccloud/alarm-api/internal/data/actual_data/ticketdb"
	"github.com/picopubliccloud/alarm-api/internal/router"
	ticketjobs "github.com/picopubliccloud/alarm-api/internal/ticketing/jobs"
	middleware "github.com/picopubliccloud/alarm-api/internal/middleware"
)

func main() {
	// Connect DBs
	db.Connect()       // topology/default DB
	ticketdb.Connect() // ticketing DB
	tdb := ticketdb.GetDB()

	// Ensure required partitions exist (ticket_updates/audit_events)
	ctx, cancel_p := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel_p()

	if err := ticketjobs.EnsurePartitions(ctx, tdb); err != nil {
		log.Fatalf("failed to ensure ticketing partitions: %v", err)
	}

	// Gin router
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// Keep your existing CORS as-is (you requested this)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// gives every request a timeout context, so DB calls can cancel.
	r.Use(middleware.RequestTimeout(2 * 60 * time.Second)) // 2 min

	// Routes
	router.RegisterRoutes(r, tdb)

	// HTTP server with timeouts + graceful shutdown
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start TLS server
	go func() {
		if err := srv.ListenAndServeTLS("ssl/cert.pem", "ssl/key.pem"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	log.Printf("listening on https://0.0.0.0:8080")

	// Wait for SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutdown signal received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	// Close DBs
	if err := ticketdb.Close(); err != nil {
		log.Printf("ticketdb close error: %v", err)
	}
	if err := db.Close(); err != nil {
		log.Printf("db close error: %v", err)
	}

	log.Printf("shutdown complete")
}