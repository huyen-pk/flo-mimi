select
    page_url,
    event_name,
    event_count,
    unique_users,
    coalesce(first_seen_at, refreshed_at) as first_seen_at,
    coalesce(last_seen_at, refreshed_at) as last_seen_at
from analytics.platform_engagement_analytics