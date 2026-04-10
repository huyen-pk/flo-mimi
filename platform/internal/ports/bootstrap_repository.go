package ports

import (
	"context"

	"platform/internal/domain/bootstrap"
)

type BootstrapRepository interface {
	Load(context.Context) (bootstrap.AppBootstrap, error)
}