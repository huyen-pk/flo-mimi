package ports

import (
    "context"

    "platform/internal/domain/bootstrap"
)

type AnalyticsRepository interface {
    LoadAnalytics(ctx context.Context) (bootstrap.Analytics, error)
}
