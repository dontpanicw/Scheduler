package main

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"os"
	"scheduler/config"
	"scheduler/internal/app"
	migrations "scheduler/pkg/migration/postgres"
)

func main() {
	// Создаём продакшен-логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	// Отложенный вызов — чтобы сбросить буферы при выходе
	defer logger.Sync()

	if err := godotenv.Load(); err != nil {
		logger.Warn("could not load .env file",
			zap.Error(err),
		)
	}
	connStr := os.Getenv("POSTGRES_CONNECTION_STRING")
	addr := os.Getenv("SERVER_ADDRESS")
	if connStr == "" && addr == "" {
		logger.Fatal("POSTGRES_CONNECTION_STRING or SERVER_ADDRESS is not set - добавьте их в .env")
	} else {
		logger.Info("config loaded",
			zap.String("postgres_conn", connStr),
			zap.String("server_addr", addr),
		)
	}

	Config := config.NewConfig(connStr, addr)

	db, err := sql.Open("pgx", Config.PgConnStr)
	if err != nil {
		logger.Fatal("error with open pgx drivet", zap.Error(err))
	}
	defer db.Close()

	// применяем миграции
	if err := migrations.Migrate(db); err != nil {
		logger.Fatal("error with create migrations", zap.Error(err))
	}

	logger.Info("Migrations applied successfully")

	if err := app.Start(Config, logger); err != nil {
		panic(err)
	}
}
