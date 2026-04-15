package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"platform/internal/domain/interactions"
	"platform/internal/telemetry"
)

type HTTPEventGateway struct {
	baseURL string
	client  *http.Client
}

func NewHTTPEventGateway(baseURL string, client *http.Client) *HTTPEventGateway {
	return &HTTPEventGateway{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (g *HTTPEventGateway) PublishInteraction(ctx context.Context, command interactions.Command) (interactions.PublishResult, error) {
	result := interactions.PublishResult{Published: make([]string, 0, 2)}

	if shouldPublishAnalytics(command) {
		if err := g.postJSON(ctx, "/events/analytics", analyticsPayload(command)); err != nil {
			return interactions.PublishResult{}, fmt.Errorf("publish analytics event: %w", err)
		}
		result.Published = append(result.Published, "analytics")
	}

	if shouldPublishEmail(command) {
		if err := g.postJSON(ctx, "/events/email", emailPayload(command)); err != nil {
			return interactions.PublishResult{}, fmt.Errorf("publish campaign event: %w", err)
		}
		result.Published = append(result.Published, "email")
	}

	return result, nil
}

func (g *HTTPEventGateway) postJSON(ctx context.Context, path string, body map[string]any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	telemetry.InjectHeaders(ctx, request.Header)

	startedAt := time.Now()
	response, err := g.client.Do(request)
	if err != nil {
		telemetry.ObserveOutboundHTTP("platform", "event-gateway", http.MethodPost, path, "error", time.Since(startedAt))
		return fmt.Errorf("execute request: %w", err)
	}
	defer response.Body.Close()
	telemetry.ObserveOutboundHTTP("platform", "event-gateway", http.MethodPost, path, strconv.Itoa(response.StatusCode), time.Since(startedAt))

	if response.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("unexpected status %s: %s", response.Status, strings.TrimSpace(string(bodyBytes)))
	}

	return nil
}

func analyticsPayload(command interactions.Command) map[string]any {
	return map[string]any{
		"session_id": command.SessionID,
		"user_id":    command.UserID,
		"event_name": slugify(command.Surface) + "." + slugify(command.Action),
		"page_url":   routePath(command.Route),
		"payload": map[string]any{
			"surface":      command.Surface,
			"action":       command.Action,
			"subject_type": command.SubjectType,
			"subject_id":   command.SubjectID,
			"campaign_id":  command.CampaignID,
			"metadata":     command.Metadata,
		},
	}
}

func emailPayload(command interactions.Command) map[string]any {
	return map[string]any{
		"campaign_id":  firstNonEmpty(command.CampaignID, command.SubjectID, "platform-ui"),
		"recipient_id": command.RecipientID,
		"event_type":   "ui_" + slugify(command.Action),
		"payload": map[string]any{
			"route":        routePath(command.Route),
			"surface":      command.Surface,
			"subject_type": command.SubjectType,
			"subject_id":   command.SubjectID,
			"metadata":     command.Metadata,
		},
	}
}

func shouldPublishAnalytics(command interactions.Command) bool {
	return command.EventKind != "email"
}

func shouldPublishEmail(command interactions.Command) bool {
	switch command.EventKind {
	case "email", "both":
		return true
	case "analytics":
		return false
	default:
		return command.SubjectType == "campaign" || command.CampaignID != ""
	}
}

func routePath(route string) string {
	route = strings.TrimSpace(route)
	if route == "" || route == "/" {
		return "/dashboard"
	}
	if strings.HasPrefix(route, "/") {
		return route
	}
	return "/" + route
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
