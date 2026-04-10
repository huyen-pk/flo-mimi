package bootstrap

type AppBootstrap struct {
	Brand       Brand        `json:"brand"`
	Session     Session      `json:"session"`
	Dashboard   Dashboard    `json:"dashboard"`
	Campaigns   Campaigns    `json:"campaigns"`
	Subscribers Subscribers  `json:"subscribers"`
	Analytics   Analytics    `json:"analytics"`
}

type Brand struct {
	Name              string `json:"name"`
	Tagline           string `json:"tagline"`
	HeroTitle         string `json:"heroTitle"`
	HeroAccent        string `json:"heroAccent"`
	HeroNote          string `json:"heroNote"`
	LastSync          string `json:"lastSync"`
	SearchPlaceholder string `json:"searchPlaceholder"`
}

type Session struct {
	Label  string `json:"label"`
	Node   string `json:"node"`
	Status string `json:"status"`
}

type Dashboard struct {
	Metrics      []Metric      `json:"metrics"`
	Performance  Performance   `json:"performance"`
	Activities   []Activity    `json:"activities"`
	Segments     []Segment     `json:"segments"`
	SecurityCard SecurityCard  `json:"securityCard"`
	BillingCard  BillingCard   `json:"billingCard"`
}

type Campaigns struct {
	Headline    string            `json:"headline"`
	Description string            `json:"description"`
	Stats       []CampaignSummary `json:"stats"`
	Items       []CampaignRecord  `json:"items"`
}

type Subscribers struct {
	Headline    string             `json:"headline"`
	Description string             `json:"description"`
	Filters     []string           `json:"filters"`
	NetworkSize string             `json:"networkSize"`
	Items       []SubscriberRecord `json:"items"`
}

type Analytics struct {
	Headline    string            `json:"headline"`
	Description string            `json:"description"`
	Pipelines   []PipelineStage   `json:"pipelines"`
	Signals     []AnalyticsSignal `json:"signals"`
}

type Metric struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Value  string `json:"value"`
	Delta  string `json:"delta"`
	Accent string `json:"accent"`
	Icon   string `json:"icon"`
	Detail string `json:"detail"`
}

type Performance struct {
	Title      string     `json:"title"`
	Modes      []string   `json:"modes"`
	ActiveMode string     `json:"activeMode"`
	Bars       []ChartBar `json:"bars"`
}

type ChartBar struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Height   int    `json:"height"`
	Emphasis bool   `json:"emphasis"`
}

type Activity struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	At          string `json:"at"`
	Tone        string `json:"tone"`
}

type Segment struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type SecurityCard struct {
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type BillingCard struct {
	Eyebrow string `json:"eyebrow"`
	Title   string `json:"title"`
	Date    string `json:"date"`
	Action  string `json:"action"`
}

type CampaignSummary struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Value    string `json:"value"`
	Detail   string `json:"detail"`
	Tone     string `json:"tone"`
	Progress int    `json:"progress"`
}

type CampaignRecord struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	Status        string `json:"status"`
	Audience      string `json:"audience"`
	AudienceLabel string `json:"audienceLabel"`
	OpenRate      string `json:"openRate"`
	ClickRate     string `json:"clickRate"`
	ActionLabel   string `json:"actionLabel"`
	Tone          string `json:"tone"`
}

type SubscriberRecord struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	SecurityStatus   string   `json:"securityStatus"`
	AssignedSegments []string `json:"assignedSegments"`
	FilterTags       []string `json:"filterTags"`
	LastInteraction  string   `json:"lastInteraction"`
	Tone             string   `json:"tone"`
}

type PipelineStage struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type AnalyticsSignal struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
	EventKind   string `json:"eventKind"`
	SubjectType string `json:"subjectType"`
	SubjectID   string `json:"subjectId"`
}