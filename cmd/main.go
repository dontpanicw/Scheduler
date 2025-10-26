package main

import (
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"log"
	"os"
	"scheduler/config"
	"scheduler/internal/app"
	migrations "scheduler/pkg/migration/postgres"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: could not load .env file: %v", err)
	}
	connStr := os.Getenv("POSTGRES_CONNECTION_STRING")
	addr := os.Getenv("SERVER_ADDRESS")
	if connStr == "" && addr == "" {
		fmt.Println("POSTGRES_CONNECTION_STRING or SERVER_ADDRESS is not set - добавьте их в .env")
	} else {
		fmt.Println("Connection string:", connStr, "; address:", addr)
	}

	Config := config.NewConfig(connStr, addr)

	db, err := sql.Open("pgx", Config.PgConnStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// применяем миграции
	if err := migrations.Migrate(db); err != nil {
		log.Fatal(err)
	}

	log.Println("Migrations applied successfully")

	if err := app.Start(Config); err != nil {
		panic(err)
	}
}
