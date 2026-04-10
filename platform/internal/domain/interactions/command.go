package interactions

type Command struct {
	SessionID   string         `json:"sessionId"`
	UserID      string         `json:"userId"`
	Route       string         `json:"route"`
	Surface     string         `json:"surface"`
	Action      string         `json:"action"`
	SubjectType string         `json:"subjectType"`
	SubjectID   string         `json:"subjectId"`
	CampaignID  string         `json:"campaignId"`
	RecipientID string         `json:"recipientId"`
	EventKind   string         `json:"eventKind"`
	Metadata    map[string]any `json:"metadata"`
}

type PublishResult struct {
	Published []string `json:"published"`
}

type Response struct {
	Accepted      bool     `json:"accepted"`
	Stored        bool     `json:"stored"`
	SessionID     string   `json:"sessionId"`
	CorrelationID string   `json:"correlationId"`
	CampaignID    string   `json:"campaignId,omitempty"`
	Published     []string `json:"published"`
	OccurredAt    string   `json:"occurredAt"`
}
