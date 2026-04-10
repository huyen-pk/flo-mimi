package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"platform/internal/domain/bootstrap"
	"platform/internal/domain/interactions"
)

type AppDBRepository struct {
	db          *sql.DB
	environment string
}

func NewAppDBRepository(dsn string, environment string) (*AppDBRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open appdb connection: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping appdb: %w", err)
	}

	return &AppDBRepository{
		db:          db,
		environment: environment,
	}, nil
}

func (r *AppDBRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *AppDBRepository) Load(ctx context.Context) (bootstrap.AppBootstrap, error) {
	const query = `
		select brand, session, dashboard, campaigns, subscribers, analytics
		from analytics.platform_bootstrap
		where environment = $1
	`

	var rawBrand []byte
	var rawSession []byte
	var rawDashboard []byte
	var rawCampaigns []byte
	var rawSubscribers []byte
	var rawAnalytics []byte

	if err := r.db.QueryRowContext(ctx, query, r.environment).Scan(
		&rawBrand,
		&rawSession,
		&rawDashboard,
		&rawCampaigns,
		&rawSubscribers,
		&rawAnalytics,
	); err != nil {
		return bootstrap.AppBootstrap{}, fmt.Errorf("load platform bootstrap: %w", err)
	}

	var model bootstrap.AppBootstrap
	if err := unmarshalSection(rawBrand, &model.Brand); err != nil {
		return bootstrap.AppBootstrap{}, fmt.Errorf("decode brand: %w", err)
	}
	if err := unmarshalSection(rawSession, &model.Session); err != nil {
		return bootstrap.AppBootstrap{}, fmt.Errorf("decode session: %w", err)
	}
	if err := unmarshalSection(rawDashboard, &model.Dashboard); err != nil {
		return bootstrap.AppBootstrap{}, fmt.Errorf("decode dashboard: %w", err)
	}
	if err := unmarshalSection(rawCampaigns, &model.Campaigns); err != nil {
		return bootstrap.AppBootstrap{}, fmt.Errorf("decode campaigns: %w", err)
	}
	if err := unmarshalSection(rawSubscribers, &model.Subscribers); err != nil {
		return bootstrap.AppBootstrap{}, fmt.Errorf("decode subscribers: %w", err)
	}
	if err := unmarshalSection(rawAnalytics, &model.Analytics); err != nil {
		return bootstrap.AppBootstrap{}, fmt.Errorf("decode analytics: %w", err)
	}

	return model, nil
}

func (r *AppDBRepository) Apply(ctx context.Context, command interactions.Command) (bool, error) {
	switch strings.TrimSpace(command.Action) {
	case "create-brief":
		if err := r.createCampaign(ctx, command); err != nil {
			return false, fmt.Errorf("store campaign brief: %w", err)
		}
		return true, nil
	case "launch-test-send":
		if err := r.scheduleCampaign(ctx, command); err != nil {
			return false, fmt.Errorf("update campaign draft: %w", err)
		}
		return true, nil
	case "add-subscriber":
		if err := r.createSubscriber(ctx, command); err != nil {
			return false, fmt.Errorf("store subscriber: %w", err)
		}
		return true, nil
	default:
		return false, nil
	}
}

func (r *AppDBRepository) createCampaign(ctx context.Context, command interactions.Command) error {
	campaignID := firstNonEmpty(strings.TrimSpace(command.CampaignID), strings.TrimSpace(command.SubjectID))
	if campaignID == "" {
		return fmt.Errorf("campaign ID is required")
	}

	displayOrder, err := r.lookupDisplayOrder(ctx, "select display_order from analytics.platform_campaign where environment = $1 and campaign_id = $2", campaignID)
	if err != nil {
		return err
	}
	if displayOrder == 0 {
		displayOrder, err = r.nextDisplayOrder(ctx, "select coalesce(max(display_order), 0) + 1 from analytics.platform_campaign where environment = $1")
		if err != nil {
			return err
		}
	}

	title := metadataString(command.Metadata, "title", defaultCampaignTitle(command.Action))
	summary := metadataString(command.Metadata, "summary", "Created from the operator console just now")
	status := metadataString(command.Metadata, "status", "Draft")
	audienceCount := metadataInt(command.Metadata, "audienceCount", 6400)
	audienceLabel := metadataString(command.Metadata, "audienceLabel", "Curated audience pending review")
	actionLabel := metadataString(command.Metadata, "actionLabel", "Launch test send")
	tone := metadataString(command.Metadata, "tone", "tertiary")

	if _, err := r.db.ExecContext(
		ctx,
		`
		insert into analytics.platform_campaign (
			environment,
			campaign_id,
			title,
			summary,
			status,
			audience_count,
			audience_label,
			action_label,
			tone,
			display_order
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		on conflict (environment, campaign_id) do update
		set title = excluded.title,
			summary = excluded.summary,
			status = excluded.status,
			audience_count = excluded.audience_count,
			audience_label = excluded.audience_label,
			action_label = excluded.action_label,
			tone = excluded.tone
		`,
		r.environment,
		campaignID,
		title,
		summary,
		status,
		audienceCount,
		audienceLabel,
		actionLabel,
		tone,
		displayOrder,
	); err != nil {
		return fmt.Errorf("upsert campaign: %w", err)
	}

	if _, err := r.db.ExecContext(
		ctx,
		`
		insert into analytics.platform_campaign_analytics (
			environment,
			campaign_id,
			delivered_recipients,
			open_recipients,
			click_recipients,
			bounce_recipients,
			refreshed_at
		) values ($1, $2, 0, 0, 0, 0, now())
		on conflict (environment, campaign_id) do nothing
		`,
		r.environment,
		campaignID,
	); err != nil {
		return fmt.Errorf("seed campaign analytics: %w", err)
	}

	return nil
}

func (r *AppDBRepository) scheduleCampaign(ctx context.Context, command interactions.Command) error {
	campaignID := firstNonEmpty(strings.TrimSpace(command.CampaignID), strings.TrimSpace(command.SubjectID))
	if campaignID == "" {
		return fmt.Errorf("campaign ID is required")
	}

	result, err := r.db.ExecContext(
		ctx,
		`
		update analytics.platform_campaign
		set status = 'Scheduled',
			summary = 'Test send queued just now',
			action_label = 'Inspect performance',
			tone = 'primary'
		where environment = $1
		  and campaign_id = $2
		`,
		r.environment,
		campaignID,
	)
	if err != nil {
		return fmt.Errorf("update campaign: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect campaign update result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("campaign %q was not found", campaignID)
	}

	return nil
}

func (r *AppDBRepository) createSubscriber(ctx context.Context, command interactions.Command) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	subscriberID := resolveSubscriberID(command)
	name := metadataString(command.Metadata, "name", "Curated Contact")
	email := metadataString(command.Metadata, "email", generatedSubscriberEmail(subscriberID))
	securityStatus := metadataString(command.Metadata, "securityStatus", "Encrypted")
	tone := metadataString(command.Metadata, "tone", "primary")
	assignedSegments := metadataStrings(command.Metadata, "assignedSegments", []string{"Priority Review"})
	filterTags := metadataStrings(command.Metadata, "filterTags", []string{"Newly Verified"})

	displayOrder, err := r.lookupDisplayOrderTx(ctx, tx, "select display_order from analytics.platform_subscriber where environment = $1 and subscriber_id = $2", subscriberID)
	if err != nil {
		return err
	}
	if displayOrder == 0 {
		displayOrder, err = r.nextDisplayOrderTx(ctx, tx, "select coalesce(max(display_order), 0) + 1 from analytics.platform_subscriber where environment = $1")
		if err != nil {
			return err
		}
	}

	if _, err = tx.ExecContext(
		ctx,
		`
		insert into analytics.platform_subscriber (
			environment,
			subscriber_id,
			name,
			email,
			security_status,
			created_at,
			last_interaction_at,
			tone,
			is_featured,
			display_order
		) values ($1, $2, $3, $4, $5, now(), now(), $6, true, $7)
		on conflict (environment, subscriber_id) do update
		set name = excluded.name,
			email = excluded.email,
			security_status = excluded.security_status,
			last_interaction_at = excluded.last_interaction_at,
			tone = excluded.tone,
			is_featured = excluded.is_featured
		`,
		r.environment,
		subscriberID,
		name,
		email,
		securityStatus,
		tone,
		displayOrder,
	); err != nil {
		return fmt.Errorf("upsert subscriber: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `delete from analytics.platform_subscriber_segment where environment = $1 and subscriber_id = $2`, r.environment, subscriberID); err != nil {
		return fmt.Errorf("clear subscriber segments: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `delete from analytics.platform_filter_assignment where environment = $1 and subscriber_id = $2`, r.environment, subscriberID); err != nil {
		return fmt.Errorf("clear subscriber filters: %w", err)
	}

	for index, segment := range assignedSegments {
		if _, err = tx.ExecContext(
			ctx,
			`insert into analytics.platform_subscriber_segment (environment, subscriber_id, segment_label, display_order) values ($1, $2, $3, $4)`,
			r.environment,
			subscriberID,
			segment,
			index+1,
		); err != nil {
			return fmt.Errorf("insert subscriber segment: %w", err)
		}
	}

	for _, filterLabel := range filterTags {
		if err = r.ensureFilterExistsTx(ctx, tx, filterLabel); err != nil {
			return err
		}
		if _, err = tx.ExecContext(
			ctx,
			`insert into analytics.platform_filter_assignment (environment, subscriber_id, filter_label) values ($1, $2, $3) on conflict do nothing`,
			r.environment,
			subscriberID,
			filterLabel,
		); err != nil {
			return fmt.Errorf("insert subscriber filter assignment: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *AppDBRepository) ensureFilterExistsTx(ctx context.Context, tx *sql.Tx, label string) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return nil
	}

	existingOrder, err := r.lookupDisplayOrderTx(ctx, tx, "select display_order from analytics.platform_filter where environment = $1 and filter_label = $2", label)
	if err != nil {
		return err
	}
	if existingOrder != 0 {
		return nil
	}

	displayOrder, err := r.nextDisplayOrderTx(ctx, tx, "select coalesce(max(display_order), 0) + 1 from analytics.platform_filter where environment = $1")
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`insert into analytics.platform_filter (environment, filter_label, display_order) values ($1, $2, $3) on conflict do nothing`,
		r.environment,
		label,
		displayOrder,
	); err != nil {
		return fmt.Errorf("insert filter %q: %w", label, err)
	}

	return nil
}

func (r *AppDBRepository) lookupDisplayOrder(ctx context.Context, query string, identifier string) (int, error) {
	var displayOrder int
	err := r.db.QueryRowContext(ctx, query, r.environment, identifier).Scan(&displayOrder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lookup display order: %w", err)
	}
	return displayOrder, nil
}

func (r *AppDBRepository) lookupDisplayOrderTx(ctx context.Context, tx *sql.Tx, query string, identifier string) (int, error) {
	var displayOrder int
	err := tx.QueryRowContext(ctx, query, r.environment, identifier).Scan(&displayOrder)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lookup display order: %w", err)
	}
	return displayOrder, nil
}

func (r *AppDBRepository) nextDisplayOrder(ctx context.Context, query string) (int, error) {
	var displayOrder int
	if err := r.db.QueryRowContext(ctx, query, r.environment).Scan(&displayOrder); err != nil {
		return 0, fmt.Errorf("select next display order: %w", err)
	}
	return displayOrder, nil
}

func (r *AppDBRepository) nextDisplayOrderTx(ctx context.Context, tx *sql.Tx, query string) (int, error) {
	var displayOrder int
	if err := tx.QueryRowContext(ctx, query, r.environment).Scan(&displayOrder); err != nil {
		return 0, fmt.Errorf("select next display order: %w", err)
	}
	return displayOrder, nil
}

func defaultCampaignTitle(action string) string {
	base := strings.TrimSpace(action)
	if base == "" {
		base = "campaign-brief"
	}
	return strings.Title(strings.ReplaceAll(base, "-", " "))
}

func resolveSubscriberID(command interactions.Command) string {
	for _, candidate := range []string{
		strings.TrimSpace(command.SubjectID),
		metadataString(command.Metadata, "subscriberId", ""),
		metadataString(command.Metadata, "email", ""),
		metadataString(command.Metadata, "name", ""),
	} {
		if candidate == "" || candidate == "new-subscriber" {
			continue
		}
		if strings.Contains(candidate, "@") {
			parts := strings.Split(candidate, "@")
			candidate = parts[0]
		}
		candidate = slugify(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return buildEntityID("subscriber")
}

func generatedSubscriberEmail(subscriberID string) string {
	return subscriberID + "@curator.local"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func metadataString(metadata map[string]any, key string, fallback string) string {
	if metadata == nil {
		return fallback
	}
	value, ok := metadata[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return strings.TrimSpace(text)
}

func metadataInt(metadata map[string]any, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}
	switch value := metadata[key].(type) {
	case float64:
		return int(value)
	case float32:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	default:
		return fallback
	}
}

func metadataStrings(metadata map[string]any, key string, fallback []string) []string {
	if metadata == nil {
		return fallback
	}
	rawValues, ok := metadata[key]
	if !ok {
		return fallback
	}

	values, ok := rawValues.([]any)
	if !ok {
		return fallback
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			result = append(result, text)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func buildEntityID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, time.Now().UTC().Format("20060102150405"))
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

func unmarshalSection[T any](raw []byte, destination *T) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty section")
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		return err
	}
	return nil
}
