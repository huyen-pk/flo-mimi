package analytics

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

	"platform/internal/domain/bootstrap"
)

type TrinoRepository struct {
	baseURL string
	user    string
	client  *http.Client
}

func NewTrinoRepository(host string, port string, user string) (*TrinoRepository, error) {
	if host == "" {
		host = "trino"
	}
	if port == "" {
		port = "8080"
	}
	if user == "" {
		user = "platform"
	}

	return &TrinoRepository{
		baseURL: fmt.Sprintf("http://%s:%s", host, port),
		user:    user,
		client:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (r *TrinoRepository) LoadAnalytics(ctx context.Context) (bootstrap.Analytics, error) {
	rows, err := r.query(ctx, `
		select
			coalesce(sum(open_events), 0) as open_events,
			coalesce(sum(click_events), 0) as click_events,
			coalesce(sum(delivered_events), 0) as delivered_events
		from clickhouse.serving.campaign_performance
	`)
	if err != nil {
		return bootstrap.Analytics{}, err
	}

	var openEvents, clickEvents, deliveredEvents int64
	if len(rows) > 0 {
		row := rows[0]
		if len(row) >= 3 {
			openEvents = int64FromNumber(row[0])
			clickEvents = int64FromNumber(row[1])
			deliveredEvents = int64FromNumber(row[2])
		}
	}

	return bootstrap.Analytics{
		Headline:    "Real-time Analytics",
		Description: fmt.Sprintf("Open %d • Click %d • Delivered %d", openEvents, clickEvents, deliveredEvents),
		Pipelines: []bootstrap.PipelineStage{{
			ID:          "realtime",
			Title:       "Real-time Analytics via Trino",
			Description: "Serving aggregates read from ClickHouse through Trino",
			Status:      "ok",
		}},
		Signals: []bootstrap.AnalyticsSignal{},
	}, nil
}

func (r *TrinoRepository) query(ctx context.Context, sql string) ([][]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/v1/statement", bytes.NewBufferString(strings.TrimSpace(sql)))
	if err != nil {
		return nil, fmt.Errorf("build trino request: %w", err)
	}
	req.Header.Set("X-Trino-User", r.user)
	req.Header.Set("X-Trino-Source", "platform")

	payload, err := r.do(req)
	if err != nil {
		return nil, err
	}

	rows := make([][]any, 0)
	for {
		if payload.Error != nil {
			return nil, fmt.Errorf("trino query failed: %s", payload.Error.Message)
		}
		if len(payload.Data) > 0 {
			rows = append(rows, payload.Data...)
		}
		if payload.NextURI == "" {
			break
		}

		nextReq, err := http.NewRequestWithContext(ctx, http.MethodGet, payload.NextURI, nil)
		if err != nil {
			return nil, fmt.Errorf("build trino next request: %w", err)
		}
		nextReq.Header.Set("X-Trino-User", r.user)
		nextReq.Header.Set("X-Trino-Source", "platform")
		payload, err = r.do(nextReq)
		if err != nil {
			return nil, err
		}
	}

	return rows, nil
}

type trinoResponse struct {
	Data    [][]any     `json:"data"`
	NextURI string      `json:"nextUri"`
	Error   *trinoError `json:"error"`
}

type trinoError struct {
	Message string `json:"message"`
}

func (r *TrinoRepository) do(req *http.Request) (trinoResponse, error) {
	resp, err := r.client.Do(req)
	if err != nil {
		return trinoResponse{}, fmt.Errorf("execute trino request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return trinoResponse{}, fmt.Errorf("trino status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return trinoResponse{}, fmt.Errorf("read trino response: %w", err)
	}
	var payload trinoResponse
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return trinoResponse{}, fmt.Errorf("decode trino response: %w", err)
	}
	return payload, nil
}

func int64FromNumber(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case json.Number:
		parsed, err := n.Int64()
		if err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseInt(n, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}