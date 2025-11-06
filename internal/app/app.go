package app

import (
	"github.com/go-chi/chi/v5"
	middleware "github.com/oapi-codegen/nethttp-middleware"
	"go.uber.org/zap"
	"scheduler/config"
	"scheduler/internal/adapter/repo/postgres"
	"scheduler/internal/cases"
	"scheduler/internal/input/http/gen"
	"scheduler/internal/input/http/handler"
)

func Start(cfg *config.Config, logger *zap.Logger) error {

	jobsRepo, err := postgres.NewJobsRepo(cfg.PgConnStr)
	if err != nil {
		logger.Fatal("failed to connect to DB:", zap.Error(err))
		return err
	}

	schedulerCase := cases.NewSchedulerCase(jobsRepo)
	schedulerHandler := handler.NewHandler(schedulerCase)

	r := chi.NewRouter()

	swagger, err := gen.GetSwagger()
	if err != nil {
		logger.Fatal("failed to load swagger", zap.Error(err))
		return err
	}
	r.Use(middleware.OapiRequestValidator(swagger))

	strictHandler := gen.NewStrictHandler(schedulerHandler, nil)
	// Регистрируем в chi роутере (теперь strictHandler — это ServerInterface)
	gen.HandlerFromMux(strictHandler, r)

	logger.Info("Starting server on", zap.String("addr:", cfg.Addr))
	if err := CreateAndRunServer(r, cfg.Addr); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
	return nil
}
