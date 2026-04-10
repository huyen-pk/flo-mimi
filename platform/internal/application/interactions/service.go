package interactions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"platform/internal/domain/interactions"
	"platform/internal/ports"
)

type Service struct {
	sink  ports.EventSink
	store ports.MutationStore
	now   func() time.Time
}

func NewService(sink ports.EventSink, store ports.MutationStore) Service {
	return Service{sink: sink, store: store, now: time.Now}
}

func (s Service) Record(ctx context.Context, command interactions.Command) (interactions.Response, error) {
	if strings.TrimSpace(command.Surface) == "" {
		return interactions.Response{}, fmt.Errorf("surface is required")
	}
	if strings.TrimSpace(command.Action) == "" {
		return interactions.Response{}, fmt.Errorf("action is required")
	}

	normalized := command
	if normalized.SessionID == "" {
		normalized.SessionID = newID("session")
	}
	if normalized.UserID == "" {
		normalized.UserID = "platform-operator"
	}
	if normalized.Route == "" {
		normalized.Route = "dashboard"
	}
	if normalized.EventKind == "" {
		normalized.EventKind = "auto"
	}
	if normalized.Metadata == nil {
		normalized.Metadata = map[string]any{}
	}
	if requiresCampaignIdentity(normalized) && normalized.CampaignID == "" {
		normalized.CampaignID = deriveCampaignID(normalized, s.now())
	}
	if normalized.Action == "create-brief" && isCampaignPlaceholder(normalized.SubjectID) {
		normalized.SubjectID = normalized.CampaignID
	}

	stored := false
	if s.store != nil {
		applied, err := s.store.Apply(ctx, normalized)
		if err != nil {
			return interactions.Response{}, err
		}
		stored = applied
	}

	result, err := s.sink.PublishInteraction(ctx, normalized)
	if err != nil {
		return interactions.Response{}, err
	}

	return interactions.Response{
		Accepted:      true,
		Stored:        stored,
		SessionID:     normalized.SessionID,
		CorrelationID: newID("trace"),
		CampaignID:    normalized.CampaignID,
		Published:     result.Published,
		OccurredAt:    s.now().UTC().Format(time.RFC3339),
	}, nil
}

func requiresCampaignIdentity(command interactions.Command) bool {
	return command.EventKind == "email" || command.EventKind == "both" || command.SubjectType == "campaign" || command.Action == "create-brief"
}

func deriveCampaignID(command interactions.Command, now time.Time) string {
	if campaignID := strings.TrimSpace(command.CampaignID); campaignID != "" {
		return campaignID
	}
	if !isCampaignPlaceholder(command.SubjectID) {
		return strings.TrimSpace(command.SubjectID)
	}
	return buildCampaignID(command.Action, now)
}

func isCampaignPlaceholder(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "new-brief", "new-campaign":
		return true
	default:
		return false
	}
}

func buildCampaignID(action string, now time.Time) string {
	return fmt.Sprintf("%s-%s", slugify(action), now.UTC().Format("20060102150405"))
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastHyphen := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastHyphen = false
		case !lastHyphen:
			builder.WriteRune('-')
			lastHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func newID(prefix string) string {
	buffer := make([]byte, 6)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buffer)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
