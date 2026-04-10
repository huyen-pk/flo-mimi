package ports

import (
	"context"

	"platform/internal/domain/interactions"
)

type MutationStore interface {
	Apply(context.Context, interactions.Command) (bool, error)
}
