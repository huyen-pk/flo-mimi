package bootstrap

import (
	"context"
	"log/slog"

	"platform/internal/domain/bootstrap"
	"platform/internal/ports"
)

type Service struct {
	repository    ports.BootstrapRepository
	analyticsRepo ports.AnalyticsRepository
}

func NewService(repository ports.BootstrapRepository, analytics ports.AnalyticsRepository) Service {
	return Service{repository: repository, analyticsRepo: analytics}
}

func (s Service) Load(ctx context.Context) (bootstrap.AppBootstrap, error) {
	model, err := s.repository.Load(ctx)
	if err != nil {
		return model, err
	}

	if s.analyticsRepo != nil {
		analytics, err := s.analyticsRepo.LoadAnalytics(ctx)
		if err != nil {
			slog.Default().Warn("load analytics from trino", "error", err)
		} else {
			model.Analytics = analytics
		}
	}

	return model, nil
}

func (s Service) LoadAnalytics(ctx context.Context) (bootstrap.Analytics, error) {
	if s.analyticsRepo == nil {
		return bootstrap.Analytics{}, nil
	}
	return s.analyticsRepo.LoadAnalytics(ctx)
}
