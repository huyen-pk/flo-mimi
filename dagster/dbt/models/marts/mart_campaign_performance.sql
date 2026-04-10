select
    campaign_id,
    delivered_recipients as delivered_events,
    open_recipients as open_events,
    click_recipients as click_events,
    coalesce(first_seen_at, refreshed_at) as first_seen_at,
    coalesce(last_seen_at, refreshed_at) as last_seen_at
from analytics.platform_campaign_analytics