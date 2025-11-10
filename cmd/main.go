package main

import (
	"database/sql"
	"go.uber.org/zap"
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

	Config, err := config.NewConfig(logger)
	if err != nil {
		logger.Fatal("error creating config", zap.Error(err))
	}

	db, err := sql.Open("pgx", Config.PG.DSN)
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
