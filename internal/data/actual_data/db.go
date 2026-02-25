// internal/data/actual_data/db.go
package db

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
	// export TOPOLOGY_DB_DSN while running the main.go
	dsn := strings.TrimSpace(os.Getenv("TOPOLOGY_DB_DSN"))
	if dsn == "" {
		log.Fatal("missing env TOPOLOGY_DB_DSN")
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open topology db: %v", err)
	}

	// sensible pool defaults (tune later)
	DB.SetMaxOpenConns(50)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(30 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)

	if err := DB.Ping(); err != nil {
		log.Fatalf("failed to ping topology db: %v", err)
	}
}

func Close() error {
	if DB == nil {
		return nil
	}
	return DB.Close()
}
