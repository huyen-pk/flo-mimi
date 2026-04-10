package ports

import (
	"context"

	"platform/internal/domain/interactions"
)

type EventSink interface {
	PublishInteraction(context.Context, interactions.Command) (interactions.PublishResult, error)
}