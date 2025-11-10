package app

import (
	"context"
	"net/http"
	"scheduler/internal/adapter/publisher"
	"scheduler/internal/adapter/subscriber"

	//"github.com/go-chi/chi/v5"
	//middleware "github.com/oapi-codegen/nethttp-middleware"
	"go.uber.org/zap"
	"scheduler/config"

	"scheduler/internal/adapter/repo/postgres"
	"scheduler/internal/cases"
	"scheduler/internal/input/http/gen"
	"scheduler/internal/input/http/handler"
)

func Start(cfg config.Config, logger *zap.Logger) error {

	pgPool, err := postgres.NewPostgresPool(context.Background(), cfg.PG)

	jobsRepo := postgres.NewJobsRepo(pgPool)

	// Create NATS JetStream publisher
	pub, err := publisher.NewNATSJobPublisher(context.Background(), logger, cfg.NatsURL)
	if err != nil {
		// Log error but continue with nil publisher (graceful degradation)
		logger.Warn("Failed to create NATS publisher, continuing without publisher", zap.Error(err))
	}

	schedulerCase := cases.NewSchedulerCase(jobsRepo, pub, cfg.SchedulerInterval, logger)
	srv := handler.NewServer(schedulerCase)

	ctx := context.Background()
	go func() {
		if err := schedulerCase.Start(ctx); err != nil {
			logger.Error("Scheduler tick loop failed", zap.Error(err))
		}
	}()

	// Create and start NATS completion subscriber
	completionSub, err := subscriber.NewNATSCompletionSubscriber(ctx, logger, cfg.NatsURL)
	if err != nil {
		logger.Warn("Failed to create NATS completion subscriber, continuing without subscriber", zap.Error(err))
	} else {
		if err := completionSub.Subscribe(ctx, func(ctx context.Context, completion subscriber.JobCompletion) error {
			return schedulerCase.HandleJobCompletion(ctx, completion.JobID, completion.Status, completion.FinishedAt)
		}); err != nil {
			logger.Warn("Failed to subscribe to completion messages", zap.Error(err))
		}
	}

	h := gen.NewStrictHandler(srv, nil)
	r := gen.HandlerWithOptions(h, gen.ChiServerOptions{})

	return http.ListenAndServe(cfg.HTTPPort, r)
}
