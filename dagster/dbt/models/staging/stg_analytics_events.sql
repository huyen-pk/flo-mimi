select
    md5(environment || ':' || page_url || ':' || event_name) as event_id,
    null::text as session_id,
    null::text as user_id,
    event_name,
    page_url,
    coalesce(last_seen_at, refreshed_at, now()) as occurred_at,
    jsonb_build_object(
        'environment', environment,
        'event_count', event_count,
        'unique_users', unique_users,
        'source', 'platform_engagement_analytics'
    ) as payload
from analytics.platform_engagement_analytics