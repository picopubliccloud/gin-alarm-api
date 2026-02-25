package ticketdb

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
	var err error

	// export TICKETING_DB_DSN while running the main.go
	dsn := os.Getenv("TICKETING_DB_DSN")
	if dsn == "" {
		log.Fatal("missing env TICKETING_DB_DSN")
	}

	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open ticketing DB: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to ping ticketing DB: %v", err)
	}

	DB.SetMaxOpenConns(50)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(30 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)

	fmt.Println("Connected to ticketing DB successfully")
}

func GetDB() *sql.DB {
	return DB
}

func Close() error {
	if DB == nil {
		return nil
	}
	return DB.Close()
}
