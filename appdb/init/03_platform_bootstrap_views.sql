create or replace function analytics.platform_format_integer(value bigint)
returns text
language sql
immutable
as $$
    select trim(to_char(coalesce(value, 0), 'FM999,999,999,990'));
$$;

create or replace function analytics.platform_format_compact(value bigint)
returns text
language sql
immutable
as $$
    select case
        when value is null then '0'
        when value >= 1000000 then trim(to_char(value / 1000000.0, 'FM990.0')) || 'm'
        when value >= 1000 then trim(to_char(value / 1000.0, 'FM990.0')) || 'k'
        else analytics.platform_format_integer(value)
    end;
$$;

create or replace function analytics.platform_format_percent(value numeric)
returns text
language sql
immutable
as $$
    select case
        when value is null then null
        else trim(to_char(value, 'FM990.0')) || '%'
    end;
$$;

create or replace function analytics.platform_relative_time(value timestamptz)
returns text
language sql
stable
as $$
    select case
        when value is null then 'No recent activity'
        when now() - value < interval '1 minute' then 'Just now'
        when now() - value < interval '1 hour' then floor(extract(epoch from now() - value) / 60)::int || ' minutes ago'
        when now() - value < interval '1 day' then floor(extract(epoch from now() - value) / 3600)::int || ' hours ago'
        when now() - value < interval '7 days' then floor(extract(epoch from now() - value) / 86400)::int || ' days ago'
        else to_char(value, 'Mon DD, YYYY')
    end;
$$;

create or replace function analytics.platform_last_sync_label(value timestamptz)
returns text
language sql
stable
as $$
    select 'Last analytics sync: ' || lower(analytics.platform_relative_time(value));
$$;

create or replace view analytics.platform_campaign_live_metrics_view as
select
    environment,
    campaign_id,
    coalesce(delivered_recipients, 0)::bigint as delivered_recipients,
    coalesce(open_recipients, 0)::bigint as open_recipients,
    coalesce(click_recipients, 0)::bigint as click_recipients,
    coalesce(bounce_recipients, 0)::bigint as bounce_recipients,
    case
        when coalesce(delivered_recipients, 0) > 0 then round(open_recipients * 100.0 / delivered_recipients, 1)
        else null
    end as open_rate,
    case
        when coalesce(delivered_recipients, 0) > 0 then round(click_recipients * 100.0 / delivered_recipients, 1)
        else null
    end as click_rate,
    case
        when coalesce(delivered_recipients, 0) + coalesce(bounce_recipients, 0) > 0 then round(delivered_recipients * 100.0 / (delivered_recipients + bounce_recipients), 2)
        else null
    end as deliverability_rate,
    first_seen_at,
    last_seen_at,
    refreshed_at
from analytics.platform_campaign_analytics;

create or replace view analytics.platform_dashboard_metrics_view as
with environments as (
    select environment
    from analytics.platform_environment
),
subscriber_stats as (
    select
        environment,
        count(*)::bigint as total_subscribers,
        count(*) filter (where created_at >= date_trunc('month', now()))::bigint as new_this_month
    from analytics.platform_subscriber
    group by environment
),
email_stats as (
    select
        environment,
        sum(delivered_recipients)::bigint as delivered_recipients,
        sum(open_recipients)::bigint as open_recipients,
        sum(click_recipients)::bigint as click_recipients
    from analytics.platform_campaign_analytics
    group by environment
),
source_metrics as (
    select
        environments.environment,
        coalesce(subscriber_stats.total_subscribers, 0) as total_subscribers,
        coalesce(subscriber_stats.new_this_month, 0) as new_this_month,
        coalesce(email_stats.delivered_recipients, 0) as delivered_recipients,
        coalesce(email_stats.open_recipients, 0) as open_recipients,
        coalesce(email_stats.click_recipients, 0) as click_recipients,
        case
            when coalesce(email_stats.delivered_recipients, 0) > 0 then round(email_stats.open_recipients * 100.0 / email_stats.delivered_recipients, 1)
            else 0
        end as open_rate,
        case
            when coalesce(email_stats.delivered_recipients, 0) > 0 then round(email_stats.click_recipients * 100.0 / email_stats.delivered_recipients, 1)
            else 0
        end as click_rate
    from environments
    left join subscriber_stats
        on subscriber_stats.environment = environments.environment
    left join email_stats
        on email_stats.environment = environments.environment
)
select
    environment,
    'total-subscribers' as id,
    'Total Subscribers' as label,
    analytics.platform_format_integer(total_subscribers) as value,
    '+' || trim(to_char(round(new_this_month * 100.0 / greatest(total_subscribers, 1), 1), 'FM990.0')) || '% this month' as delta,
    'success' as accent,
    'group' as icon,
    'CRUD-synced subscriber registry' as detail,
    1 as display_order
from source_metrics
union all
select
    environment,
    'avg-open-rate',
    'Avg. Open Rate',
    analytics.platform_format_percent(open_rate),
    analytics.platform_format_integer(open_recipients) || ' opens from consolidated campaign analytics',
    'primary',
    'mail',
    'Refreshed from campaign aggregates',
    2
from source_metrics
union all
select
    environment,
    'click-through-rate',
    'Click Through Rate',
    analytics.platform_format_percent(click_rate),
    analytics.platform_format_integer(click_recipients) || ' clicks from consolidated campaign analytics',
    'muted',
    'ads_click',
    'Refreshed from campaign aggregates',
    3
from source_metrics;

create or replace view analytics.platform_campaign_summary_view as
with campaign_totals as (
    select
        environment,
        sum(audience_count)::bigint as total_reach,
        count(*) filter (where status = 'Sent')::int as sent_campaigns
    from analytics.platform_campaign
    group by environment
),
performance as (
    select
        campaign.environment,
        avg(metrics.open_rate) filter (where campaign.status = 'Sent' and metrics.open_rate is not null) as avg_open_rate,
        sum(coalesce(metrics.delivered_recipients, 0))::bigint as delivered_recipients,
        sum(coalesce(metrics.bounce_recipients, 0))::bigint as bounce_recipients,
        case
            when sum(coalesce(metrics.delivered_recipients, 0) + coalesce(metrics.bounce_recipients, 0)) > 0 then round(sum(coalesce(metrics.delivered_recipients, 0)) * 100.0 / sum(coalesce(metrics.delivered_recipients, 0) + coalesce(metrics.bounce_recipients, 0)), 2)
            else 0
        end as deliverability_rate
    from analytics.platform_campaign as campaign
    left join analytics.platform_campaign_live_metrics_view as metrics
        on metrics.environment = campaign.environment
       and metrics.campaign_id = campaign.campaign_id
    group by campaign.environment
)
select
    campaign_totals.environment,
    'total-reach' as id,
    'Total Reach' as label,
    analytics.platform_format_compact(campaign_totals.total_reach) as value,
    campaign_totals.sent_campaigns || ' sent campaigns' as detail,
    'muted' as tone,
    0 as progress,
    1 as display_order
from campaign_totals
union all
select
    performance.environment,
    'avg-engagement',
    'Avg Engagement',
    coalesce(analytics.platform_format_percent(performance.avg_open_rate), '0.0%'),
    analytics.platform_format_integer(performance.delivered_recipients) || ' deliveries consolidated',
    'primary',
    0,
    2
from performance
union all
select
    performance.environment,
    'deliverability',
    'Deliverability',
    analytics.platform_format_percent(performance.deliverability_rate),
    analytics.platform_format_integer(performance.bounce_recipients) || ' bounced addresses',
    'progress',
    greatest(0, least(100, round(performance.deliverability_rate)::int)),
    3
from performance;

create or replace view analytics.platform_dashboard_bars_view as
with environments as (
    select environment
    from analytics.platform_environment
),
day_series as (
    select
        environments.environment,
        generate_series(current_date - interval '6 days', current_date, interval '1 day')::date as bucket_date
    from environments
),
merged as (
    select
        day_series.environment,
        day_series.bucket_date,
        coalesce(performance.event_count, 0) as event_count
    from day_series
    left join analytics.platform_daily_performance as performance
        on performance.environment = day_series.environment
       and performance.bucket_date = day_series.bucket_date
),
ranked as (
    select
        environment,
        bucket_date,
        event_count,
        greatest(18, least(94, round(event_count * 94.0 / greatest(max(event_count) over (partition by environment), 1))::int)) as height,
        row_number() over (partition by environment order by event_count desc, bucket_date desc) as emphasis_rank,
        row_number() over (partition by environment order by bucket_date) as display_order
    from merged
)
select
    environment,
    trim(to_char(bucket_date, 'Dy')) as label,
    analytics.platform_format_compact(event_count) as value,
    height,
    emphasis_rank = 1 as emphasis,
    display_order,
    bucket_date,
    event_count
from ranked;

create or replace view analytics.platform_bootstrap as
with environment_rows as (
    select *
    from analytics.platform_environment
),
last_sync as (
    select
        environment_rows.environment,
        greatest(
            environment_rows.updated_at,
            coalesce((
                select max(refreshed_at)
                from analytics.platform_campaign_analytics
                where environment = environment_rows.environment
            ), '-infinity'::timestamptz),
            coalesce((
                select max(last_seen_at)
                from analytics.platform_campaign_analytics
                where environment = environment_rows.environment
            ), '-infinity'::timestamptz),
            coalesce((
                select max(refreshed_at)
                from analytics.platform_engagement_analytics
                where environment = environment_rows.environment
            ), '-infinity'::timestamptz),
            coalesce((
                select max(last_seen_at)
                from analytics.platform_engagement_analytics
                where environment = environment_rows.environment
            ), '-infinity'::timestamptz),
            coalesce((
                select max(refreshed_at)
                from analytics.platform_daily_performance
                where environment = environment_rows.environment
            ), '-infinity'::timestamptz)
        ) as synced_at
    from environment_rows
),
subscriber_assignments as (
    select
        subscriber.environment,
        subscriber.subscriber_id,
        coalesce((
            select jsonb_agg(segment.segment_label order by segment.display_order)
            from analytics.platform_subscriber_segment as segment
            where segment.environment = subscriber.environment
              and segment.subscriber_id = subscriber.subscriber_id
        ), '[]'::jsonb) as assigned_segments,
        coalesce((
            select jsonb_agg(filter_assignment.filter_label order by platform_filter.display_order)
            from analytics.platform_filter_assignment as filter_assignment
            join analytics.platform_filter as platform_filter
                on platform_filter.environment = filter_assignment.environment
               and platform_filter.filter_label = filter_assignment.filter_label
            where filter_assignment.environment = subscriber.environment
              and filter_assignment.subscriber_id = subscriber.subscriber_id
        ), '[]'::jsonb) as filter_tags
    from analytics.platform_subscriber as subscriber
)
select
    environment_rows.environment,
    jsonb_build_object(
        'name', environment_rows.brand_name,
        'tagline', environment_rows.brand_tagline,
        'heroTitle', environment_rows.hero_title,
        'heroAccent', environment_rows.hero_accent,
        'heroNote', environment_rows.hero_note,
        'lastSync', analytics.platform_last_sync_label(last_sync.synced_at),
        'searchPlaceholder', environment_rows.search_placeholder
    ) as brand,
    jsonb_build_object(
        'label', environment_rows.session_label,
        'node', environment_rows.session_node,
        'status', environment_rows.session_status
    ) as session,
    jsonb_build_object(
        'metrics', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', metric.id,
                    'label', metric.label,
                    'value', metric.value,
                    'delta', metric.delta,
                    'accent', metric.accent,
                    'icon', metric.icon,
                    'detail', metric.detail
                )
                order by metric.display_order
            )
            from analytics.platform_dashboard_metrics_view as metric
            where metric.environment = environment_rows.environment
        ), '[]'::jsonb),
        'performance', jsonb_build_object(
            'title', 'Campaign Performance',
            'modes', jsonb_build_array('Daily', 'Weekly'),
            'activeMode', 'Daily',
            'bars', coalesce((
                select jsonb_agg(
                    jsonb_build_object(
                        'label', bar.label,
                        'value', bar.value,
                        'height', bar.height,
                        'emphasis', bar.emphasis
                    )
                    order by bar.display_order
                )
                from analytics.platform_dashboard_bars_view as bar
                where bar.environment = environment_rows.environment
            ), '[]'::jsonb)
        ),
        'activities', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', activity.activity_id,
                    'title', activity.title,
                    'description', activity.description,
                    'at', analytics.platform_relative_time(activity.occurred_at),
                    'tone', activity.tone
                )
                order by activity.display_order
            )
            from analytics.platform_dashboard_activity as activity
            where activity.environment = environment_rows.environment
        ), '[]'::jsonb),
        'segments', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', segment.segment_id,
                    'label', segment.label,
                    'value', analytics.platform_format_percent(segment.percentage)
                )
                order by segment.display_order
            )
            from analytics.platform_growth_segment as segment
            where segment.environment = environment_rows.environment
        ), '[]'::jsonb),
        'securityCard', jsonb_build_object(
            'title', environment_rows.security_title,
            'status', environment_rows.security_status,
            'description', environment_rows.security_description
        ),
        'billingCard', jsonb_build_object(
            'eyebrow', environment_rows.billing_eyebrow,
            'title', environment_rows.billing_title,
            'date', environment_rows.billing_date,
            'action', environment_rows.billing_action
        )
    ) as dashboard,
    jsonb_build_object(
        'headline', environment_rows.campaigns_headline,
        'description', environment_rows.campaigns_description,
        'stats', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', summary.id,
                    'label', summary.label,
                    'value', summary.value,
                    'detail', summary.detail,
                    'tone', summary.tone,
                    'progress', summary.progress
                )
                order by summary.display_order
            )
            from analytics.platform_campaign_summary_view as summary
            where summary.environment = environment_rows.environment
        ), '[]'::jsonb),
        'items', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', campaign.campaign_id,
                    'title', campaign.title,
                    'summary', campaign.summary,
                    'status', campaign.status,
                    'audience', analytics.platform_format_integer(campaign.audience_count),
                    'audienceLabel', campaign.audience_label,
                    'openRate', case
                        when campaign.status = 'Sent' then coalesce(analytics.platform_format_percent(metrics.open_rate), '--')
                        else '--'
                    end,
                    'clickRate', case
                        when campaign.status = 'Sent' then coalesce(analytics.platform_format_percent(metrics.click_rate), '--')
                        else '--'
                    end,
                    'actionLabel', campaign.action_label,
                    'tone', campaign.tone
                )
                order by campaign.display_order
            )
            from analytics.platform_campaign as campaign
            left join analytics.platform_campaign_live_metrics_view as metrics
                on metrics.environment = campaign.environment
               and metrics.campaign_id = campaign.campaign_id
            where campaign.environment = environment_rows.environment
        ), '[]'::jsonb)
    ) as campaigns,
    jsonb_build_object(
        'headline', environment_rows.subscribers_headline,
        'description', environment_rows.subscribers_description,
        'filters', coalesce((
            select jsonb_agg(filter_label order by display_order)
            from analytics.platform_filter
            where environment = environment_rows.environment
        ), '[]'::jsonb),
        'networkSize', analytics.platform_format_integer((
            select count(*)::bigint
            from analytics.platform_subscriber
            where environment = environment_rows.environment
        )),
        'items', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', subscriber.subscriber_id,
                    'name', subscriber.name,
                    'email', subscriber.email,
                    'securityStatus', subscriber.security_status,
                    'assignedSegments', assignments.assigned_segments,
                    'filterTags', assignments.filter_tags,
                    'lastInteraction', analytics.platform_relative_time(subscriber.last_interaction_at),
                    'tone', subscriber.tone
                )
                order by subscriber.display_order
            )
            from analytics.platform_subscriber as subscriber
            left join subscriber_assignments as assignments
                on assignments.environment = subscriber.environment
               and assignments.subscriber_id = subscriber.subscriber_id
            where subscriber.environment = environment_rows.environment
              and subscriber.is_featured
        ), '[]'::jsonb)
    ) as subscribers,
    jsonb_build_object(
        'headline', environment_rows.analytics_headline,
        'description', environment_rows.analytics_description,
        'pipelines', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', pipeline.pipeline_id,
                    'title', pipeline.title,
                    'description', pipeline.description,
                    'status', pipeline.status
                )
                order by pipeline.display_order
            )
            from analytics.platform_analytics_pipeline as pipeline
            where pipeline.environment = environment_rows.environment
        ), '[]'::jsonb),
        'signals', coalesce((
            select jsonb_agg(
                jsonb_build_object(
                    'id', signal.signal_id,
                    'title', signal.title,
                    'description', signal.description,
                    'action', signal.action,
                    'eventKind', signal.event_kind,
                    'subjectType', signal.subject_type,
                    'subjectId', signal.subject_id
                )
                order by signal.display_order
            )
            from analytics.platform_analytics_signal as signal
            where signal.environment = environment_rows.environment
        ), '[]'::jsonb)
    ) as analytics
from environment_rows
join last_sync
    on last_sync.environment = environment_rows.environment;
