package interactions

import (
	"context"
	"testing"
	"time"

	domain "platform/internal/domain/interactions"
)

type stubSink struct {
	lastCommand domain.Command
	result      domain.PublishResult
}

type stubStore struct {
	lastCommand domain.Command
	applied     bool
}

func (s *stubSink) PublishInteraction(_ context.Context, command domain.Command) (domain.PublishResult, error) {
	s.lastCommand = command
	return s.result, nil
}

func (s *stubStore) Apply(_ context.Context, command domain.Command) (bool, error) {
	s.lastCommand = command
	return s.applied, nil
}

func TestRecordNormalizesCampaignInteractions(t *testing.T) {
	sink := &stubSink{result: domain.PublishResult{Published: []string{"analytics", "email"}}}
	service := Service{
		sink: sink,
		now: func() time.Time {
			return time.Date(2026, time.April, 9, 10, 11, 12, 0, time.UTC)
		},
	}

	response, err := service.Record(context.Background(), domain.Command{
		Surface:     "campaign-row",
		Action:      "launch-test-send",
		SubjectType: "campaign",
		SubjectID:   "product-unveiling-zenith",
	})
	if err != nil {
		t.Fatalf("record interaction: %v", err)
	}

	if sink.lastCommand.SessionID == "" {
		t.Fatalf("expected session ID to be generated")
	}
	if sink.lastCommand.CampaignID != "product-unveiling-zenith" {
		t.Fatalf("expected campaign ID to default from subject ID, got %q", sink.lastCommand.CampaignID)
	}
	if response.CampaignID != "product-unveiling-zenith" {
		t.Fatalf("expected response campaign ID, got %q", response.CampaignID)
	}
	if len(response.Published) != 2 {
		t.Fatalf("expected 2 published channels, got %d", len(response.Published))
	}
}

func TestRecordPersistsMutatingInteractions(t *testing.T) {
	sink := &stubSink{result: domain.PublishResult{Published: []string{"analytics", "email"}}}
	store := &stubStore{applied: true}
	service := Service{
		sink:  sink,
		store: store,
		now: func() time.Time {
			return time.Date(2026, time.April, 9, 10, 11, 12, 0, time.UTC)
		},
	}

	response, err := service.Record(context.Background(), domain.Command{
		Surface:     "campaigns-header",
		Action:      "create-brief",
		SubjectType: "campaign",
		SubjectID:   "new-brief",
	})
	if err != nil {
		t.Fatalf("record interaction: %v", err)
	}

	if response.Stored != true {
		t.Fatalf("expected stored response to be true")
	}
	if response.CampaignID == "" || response.CampaignID == "new-brief" {
		t.Fatalf("expected generated campaign ID, got %q", response.CampaignID)
	}
	if store.lastCommand.CampaignID != response.CampaignID {
		t.Fatalf("expected store to receive generated campaign ID %q, got %q", response.CampaignID, store.lastCommand.CampaignID)
	}
	if sink.lastCommand.SubjectID != response.CampaignID {
		t.Fatalf("expected subject ID to be normalized to generated campaign ID, got %q", sink.lastCommand.SubjectID)
	}
}
