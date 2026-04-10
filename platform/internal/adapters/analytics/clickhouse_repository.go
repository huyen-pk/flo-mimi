package analytics

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "time"

    "platform/internal/domain/bootstrap"
)

type ClickHouseRepository struct {
    baseURL  string
    client   *http.Client
    user     string
    password string
}

func NewClickHouseRepository(host string, port string) (*ClickHouseRepository, error) {
    if host == "" {
        host = "clickhouse"
    }
    if port == "" {
        port = "8123"
    }
    base := fmt.Sprintf("http://%s:%s", host, port)
    client := &http.Client{Timeout: 10 * time.Second}
    user := os.Getenv("CLICKHOUSE_USER")
    password := os.Getenv("CLICKHOUSE_PASSWORD")
    return &ClickHouseRepository{baseURL: base, client: client, user: user, password: password}, nil
}

func (r *ClickHouseRepository) LoadAnalytics(ctx context.Context) (bootstrap.Analytics, error) {
    q := "select sum(open_events) as open_events, sum(click_events) as click_events, sum(delivered_events) as delivered_events from serving.campaign_performance FORMAT JSON"
    respData, err := r.doQuery(ctx, q)
    if err != nil {
        return bootstrap.Analytics{}, err
    }

    var openEvents, clickEvents, deliveredEvents int64
    if len(respData) > 0 {
        row := respData[0]
        if v, ok := row["open_events"]; ok {
            openEvents = int64FromNumber(v)
        }
        if v, ok := row["click_events"]; ok {
            clickEvents = int64FromNumber(v)
        }
        if v, ok := row["delivered_events"]; ok {
            deliveredEvents = int64FromNumber(v)
        }
    }

    analytics := bootstrap.Analytics{
        Headline:    "Real-time Analytics",
        Description: fmt.Sprintf("Open %d • Click %d • Delivered %d", openEvents, clickEvents, deliveredEvents),
        Pipelines: []bootstrap.PipelineStage{
            {
                ID:          "realtime",
                Title:       "Real-time ClickHouse",
                Description: "Streaming analytics from Redpanda",
                Status:      "ok",
            },
        },
        Signals: []bootstrap.AnalyticsSignal{},
    }
    return analytics, nil
}

func (r *ClickHouseRepository) doQuery(ctx context.Context, query string) ([]map[string]any, error) {
    u := r.baseURL + "/?query=" + url.QueryEscape(query)
    if r.user != "" {
        u = u + "&user=" + url.QueryEscape(r.user) + "&password=" + url.QueryEscape(r.password)
    }
    req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
    if err != nil {
        return nil, err
    }
    // Use query params for authentication to avoid Authorization header conflicts
    resp, err := r.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("clickhouse query status %d: %s", resp.StatusCode, string(bodyBytes))
    }
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    var parsed struct {
        Meta []map[string]any `json:"meta"`
        Data []map[string]any `json:"data"`
        Rows int              `json:"rows"`
    }
    if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
        return nil, fmt.Errorf("parse clickhouse json: %w", err)
    }
    return parsed.Data, nil
}

func int64FromNumber(v any) int64 {
    switch n := v.(type) {
    case float64:
        return int64(n)
    case int64:
        return n
    case int:
        return int64(n)
    case string:
        // ClickHouse may return numeric values as strings in JSON; parse them
        if parsed, err := strconv.ParseInt(n, 10, 64); err == nil {
            return parsed
        }
    default:
        return 0
    }
    return 0
}
