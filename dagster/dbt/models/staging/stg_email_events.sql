select
    md5(environment || ':' || campaign_id) as event_id,
    campaign_id,
    null::text as recipient_id,
    'consolidated'::text as event_type,
    coalesce(last_seen_at, refreshed_at, now()) as occurred_at,
    jsonb_build_object(
        'environment', environment,
        'delivered_events', delivered_recipients,
        'open_events', open_recipients,
        'click_events', click_recipients,
        'bounce_events', bounce_recipients,
        'source', 'platform_campaign_analytics'
    ) as payload
from analytics.platform_campaign_analytics