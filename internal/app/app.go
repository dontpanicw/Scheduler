package app

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	middleware "github.com/oapi-codegen/nethttp-middleware"
	"log"
	"scheduler/config"
	"scheduler/internal/adapter/repo/postgres"
	"scheduler/internal/cases"
	"scheduler/internal/input/http/gen"
	"scheduler/internal/input/http/handler"
)

func Start(cfg *config.Config) error {

	jobsRepo, err := postgres.NewJobsRepo(cfg.PgConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}

	schedulerCase := cases.NewSchedulerCase(jobsRepo)
	schedulerHandler := handler.NewHandler(schedulerCase)

	r := chi.NewRouter()

	swagger, err := gen.GetSwagger()
	if err != nil {
		return fmt.Errorf("failed to load swagger: %w", err)
	}
	r.Use(middleware.OapiRequestValidator(swagger))

	strictHandler := gen.NewStrictHandler(schedulerHandler, nil)
	// Регистрируем в chi роутере (теперь strictHandler — это ServerInterface)
	gen.HandlerFromMux(strictHandler, r)

	log.Printf("Starting server on %s", cfg.Addr)
	if err := CreateAndRunServer(r, fmt.Sprintf("localhost:%s", cfg.Addr)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	return nil
}
