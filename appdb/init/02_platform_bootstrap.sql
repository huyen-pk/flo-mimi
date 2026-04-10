do $$
begin
    if exists (
        select 1
        from pg_class as relation
        join pg_namespace as namespace
            on namespace.oid = relation.relnamespace
        where namespace.nspname = 'analytics'
          and relation.relname = 'platform_bootstrap'
          and relation.relkind = 'v'
    ) then
        execute 'drop view analytics.platform_bootstrap cascade';
    elsif exists (
        select 1
        from pg_class as relation
        join pg_namespace as namespace
            on namespace.oid = relation.relnamespace
        where namespace.nspname = 'analytics'
          and relation.relname = 'platform_bootstrap'
          and relation.relkind in ('r', 'p')
    ) then
        execute 'drop table analytics.platform_bootstrap cascade';
    end if;
end
$$;
drop view if exists analytics.platform_dashboard_metrics_view cascade;
drop view if exists analytics.platform_dashboard_bars_view cascade;
drop view if exists analytics.platform_campaign_summary_view cascade;
drop view if exists analytics.platform_campaign_live_metrics_view cascade;

create table if not exists analytics.platform_environment (
    environment text primary key,
    brand_name text not null,
    brand_tagline text not null,
    hero_title text not null,
    hero_accent text not null,
    hero_note text not null,
    search_placeholder text not null,
    session_label text not null,
    session_node text not null,
    session_status text not null,
    campaigns_headline text not null,
    campaigns_description text not null,
    subscribers_headline text not null,
    subscribers_description text not null,
    analytics_headline text not null,
    analytics_description text not null,
    security_title text not null,
    security_status text not null,
    security_description text not null,
    billing_eyebrow text not null,
    billing_title text not null,
    billing_date text not null,
    billing_action text not null,
    updated_at timestamptz not null default now()
);

create table if not exists analytics.platform_filter (
    environment text not null,
    filter_label text not null,
    display_order integer not null,
    primary key (environment, filter_label),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_dashboard_activity (
    environment text not null,
    activity_id text not null,
    title text not null,
    description text not null,
    occurred_at timestamptz not null,
    tone text not null,
    display_order integer not null,
    primary key (environment, activity_id),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_growth_segment (
    environment text not null,
    segment_id text not null,
    label text not null,
    percentage numeric(5,2) not null,
    display_order integer not null,
    primary key (environment, segment_id),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_campaign (
    environment text not null,
    campaign_id text not null,
    title text not null,
    summary text not null,
    status text not null,
    audience_count integer not null,
    audience_label text not null,
    action_label text not null,
    tone text not null,
    display_order integer not null,
    primary key (environment, campaign_id),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_subscriber (
    environment text not null,
    subscriber_id text not null,
    name text not null,
    email text not null,
    security_status text not null,
    created_at timestamptz not null,
    last_interaction_at timestamptz,
    tone text not null,
    is_featured boolean not null default false,
    display_order integer not null,
    primary key (environment, subscriber_id),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_subscriber_segment (
    environment text not null,
    subscriber_id text not null,
    segment_label text not null,
    display_order integer not null,
    primary key (environment, subscriber_id, segment_label),
    foreign key (environment, subscriber_id) references analytics.platform_subscriber (environment, subscriber_id) on delete cascade
);

create table if not exists analytics.platform_filter_assignment (
    environment text not null,
    subscriber_id text not null,
    filter_label text not null,
    primary key (environment, subscriber_id, filter_label),
    foreign key (environment, subscriber_id) references analytics.platform_subscriber (environment, subscriber_id) on delete cascade,
    foreign key (environment, filter_label) references analytics.platform_filter (environment, filter_label) on delete cascade
);

create table if not exists analytics.platform_analytics_pipeline (
    environment text not null,
    pipeline_id text not null,
    title text not null,
    description text not null,
    status text not null,
    display_order integer not null,
    primary key (environment, pipeline_id),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_analytics_signal (
    environment text not null,
    signal_id text not null,
    title text not null,
    description text not null,
    action text not null,
    event_kind text not null,
    subject_type text not null,
    subject_id text not null,
    display_order integer not null,
    primary key (environment, signal_id),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_campaign_analytics (
    environment text not null,
    campaign_id text not null,
    delivered_recipients bigint not null default 0,
    open_recipients bigint not null default 0,
    click_recipients bigint not null default 0,
    bounce_recipients bigint not null default 0,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    refreshed_at timestamptz not null default now(),
    primary key (environment, campaign_id),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade,
    foreign key (environment, campaign_id) references analytics.platform_campaign (environment, campaign_id) on delete cascade
);

create table if not exists analytics.platform_engagement_analytics (
    environment text not null,
    page_url text not null,
    event_name text not null,
    event_count bigint not null default 0,
    unique_users bigint not null default 0,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    refreshed_at timestamptz not null default now(),
    display_order integer not null,
    primary key (environment, page_url, event_name),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create table if not exists analytics.platform_daily_performance (
    environment text not null,
    bucket_date date not null,
    event_count bigint not null default 0,
    refreshed_at timestamptz not null default now(),
    primary key (environment, bucket_date),
    foreign key (environment) references analytics.platform_environment (environment) on delete cascade
);

create index if not exists idx_platform_subscriber_environment_featured on analytics.platform_subscriber (environment, is_featured, display_order);
create index if not exists idx_platform_subscriber_created_at on analytics.platform_subscriber (environment, created_at);
create index if not exists idx_platform_campaign_display_order on analytics.platform_campaign (environment, display_order);
create index if not exists idx_platform_campaign_analytics_refresh on analytics.platform_campaign_analytics (environment, refreshed_at desc);
create index if not exists idx_platform_engagement_analytics_refresh on analytics.platform_engagement_analytics (environment, refreshed_at desc);
create index if not exists idx_platform_daily_performance_refresh on analytics.platform_daily_performance (environment, refreshed_at desc);

delete from analytics.platform_daily_performance where environment = 'local';
delete from analytics.platform_engagement_analytics where environment = 'local';
delete from analytics.platform_campaign_analytics where environment = 'local';
delete from analytics.platform_filter_assignment where environment = 'local';
delete from analytics.platform_subscriber_segment where environment = 'local';
delete from analytics.platform_analytics_signal where environment = 'local';
delete from analytics.platform_analytics_pipeline where environment = 'local';
delete from analytics.platform_dashboard_activity where environment = 'local';
delete from analytics.platform_growth_segment where environment = 'local';
delete from analytics.platform_subscriber where environment = 'local';
delete from analytics.platform_campaign where environment = 'local';
delete from analytics.platform_filter where environment = 'local';
delete from analytics.platform_environment where environment = 'local';

insert into analytics.platform_environment (
    environment,
    brand_name,
    brand_tagline,
    hero_title,
    hero_accent,
    hero_note,
    search_placeholder,
    session_label,
    session_node,
    session_status,
    campaigns_headline,
    campaigns_description,
    subscribers_headline,
    subscribers_description,
    analytics_headline,
    analytics_description,
    security_title,
    security_status,
    security_description,
    billing_eyebrow,
    billing_title,
    billing_date,
    billing_action
)
values (
    'local',
    'The Curator',
    'Fortified Serenity',
    'Your communication',
    'Intelligence Hub.',
    'All systems secure',
    'Search secure archives...',
    'Active Session',
    'Encrypted Node: 04X-88',
    'online',
    'Campaigns',
    'Orchestrate your high-fidelity communications with fortified precision and editorial grace.',
    'Subscribers',
    'Orchestrate your network with absolute privacy and editorial precision.',
    'Interaction Telemetry',
    'Live appdb aggregates now drive the dashboard, campaign rates, and bootstrap payload while the platform reference content remains relational and seedable.',
    'DMARC Status',
    'Verified',
    'Your sending domain remains fully authenticated and protected against spoofing.',
    'Pro Plan Active',
    'Next Billing',
    'May 24, 2026',
    'Manage Subscription'
);

insert into analytics.platform_filter (environment, filter_label, display_order)
values
    ('local', 'All Contacts', 1),
    ('local', 'High Value', 2),
    ('local', 'Inactive', 3),
    ('local', 'Newly Verified', 4);

insert into analytics.platform_dashboard_activity (
    environment,
    activity_id,
    title,
    description,
    occurred_at,
    tone,
    display_order
)
values
    ('local', 'executive-brief-sent', 'Executive Briefing dispatched', 'Seeded audience cohorts are now mirrored into appdb-backed platform views.', now() - interval '18 minutes', 'primary', 1),
    ('local', 'segment-refresh', 'High Value cohort refreshed', 'Segment tags are normalized in SQL and displayed through the subscriber filters.', now() - interval '2 hours', 'success', 2),
    ('local', 'privacy-review', 'Privacy review cleared', 'The seed model is no longer compiled into the binary and can be recreated from SQL.', now() - interval '5 hours', 'muted', 3),
    ('local', 'launch-draft', 'Zenith launch draft edited', 'Campaign performance cards now read live delivery, open, and click rates from appdb.', now() - interval '1 day', 'primary', 4);

insert into analytics.platform_growth_segment (environment, segment_id, label, percentage, display_order)
values
    ('local', 'us', 'United States', 82.0, 1),
    ('local', 'uk', 'United Kingdom', 64.0, 2),
    ('local', 'sg', 'Singapore', 49.0, 3),
    ('local', 'de', 'Germany', 41.0, 4);

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
)
values
    ('local', 'spring-capital-briefing', 'Spring Capital Briefing', 'Sent Apr 08, 2026', 'Sent', 46800, 'High Value circle', 'Inspect performance', 'primary', 1),
    ('local', 'monthly-security-digest', 'Monthly Security Digest', 'Scheduled for Apr 15, 2026', 'Scheduled', 17240, 'Newly verified readers', 'Preview draft', 'muted', 2),
    ('local', 'product-unveiling-zenith', 'Product Unveiling: Zenith', 'Last edited 2 hours ago', 'Draft', 28400, 'Board list and strategic partners', 'Launch test send', 'tertiary', 3),
    ('local', 'quarterly-insight-report', 'Quarterly Insight Report', 'Sent Mar 21, 2026', 'Sent', 118000, 'Global partners', 'Export report', 'primary', 4),
    ('local', 'member-reengagement-arc', 'Member Re-engagement Arc', 'Queued for segment backfill', 'Sent', 9860, 'Inactive subscribers', 'Inspect audience', 'muted', 5);

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
)
values
    ('local', 'evelyn-sterling', 'Evelyn Sterling', 'e.sterling@private.com', 'Encrypted', now() - interval '12 days', now() - interval '24 minutes', 'primary', true, 1),
    ('local', 'marcus-aurelius-ii', 'Marcus Aurelius II', 'm.aurelius@heritage.org', 'Screening', now() - interval '148 days', now() - interval '9 days', 'muted', true, 2),
    ('local', 'julian-thorne', 'Julian Thorne', 'jthorne@global.nexus', 'Encrypted', now() - interval '43 days', now() - interval '3 days', 'primary', true, 3),
    ('local', 'amira-sterling', 'Amira Sterling', 'amira@capital.circle', 'Review', now() - interval '27 days', now() - interval '7 days', 'tertiary', true, 4),
    ('local', 'noah-vesper', 'Noah Vesper', 'nvesper@portfolio.one', 'Encrypted', now() - interval '4 days', now() - interval '42 minutes', 'primary', true, 5),
    ('local', 'lila-somerset', 'Lila Somerset', 'lila@somerset.foundation', 'Dormant', now() - interval '220 days', now() - interval '21 days', 'muted', true, 6);

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
)
select
    'local',
    format('seed-member-%05s', sequence_id),
    format('Portfolio Member %s', sequence_id),
    format('member%05s@seed.curator.local', sequence_id),
    case when sequence_id % 31 = 0 then 'Review' else 'Encrypted' end,
    case
        when sequence_id <= 1424 then now() - make_interval(days => (sequence_id % 10))
        else now() - make_interval(days => 35 + (sequence_id % 560))
    end,
    now() - make_interval(hours => 6 + (sequence_id % 480)),
    case when sequence_id % 31 = 0 then 'tertiary' else 'primary' end,
    false,
    1000 + sequence_id
from generate_series(1, 12836) as generated(sequence_id);

insert into analytics.platform_subscriber_segment (environment, subscriber_id, segment_label, display_order)
values
    ('local', 'evelyn-sterling', 'VVIP', 1),
    ('local', 'evelyn-sterling', 'Q4 Launch', 2),
    ('local', 'marcus-aurelius-ii', 'Art Collector', 1),
    ('local', 'julian-thorne', 'Board List', 1),
    ('local', 'julian-thorne', 'Europe', 2),
    ('local', 'amira-sterling', 'High Intent', 1),
    ('local', 'amira-sterling', 'VIP', 2),
    ('local', 'noah-vesper', 'New Investors', 1),
    ('local', 'noah-vesper', 'APAC', 2),
    ('local', 'lila-somerset', 'Legacy Circle', 1);

insert into analytics.platform_filter_assignment (environment, subscriber_id, filter_label)
values
    ('local', 'evelyn-sterling', 'High Value'),
    ('local', 'evelyn-sterling', 'Newly Verified'),
    ('local', 'marcus-aurelius-ii', 'Inactive'),
    ('local', 'julian-thorne', 'High Value'),
    ('local', 'amira-sterling', 'High Value'),
    ('local', 'noah-vesper', 'Newly Verified'),
    ('local', 'lila-somerset', 'Inactive');

insert into analytics.platform_analytics_pipeline (
    environment,
    pipeline_id,
    title,
    description,
    status,
    display_order
)
values
    ('local', 'ui', 'Svelte Interaction', 'Buttons, filters, row actions, and navigation emit typed commands through the embedded Go API.', 'Live', 1),
    ('local', 'seed', 'Normalized SQL Seed', 'Reference content is stored in relational appdb tables while the bootstrap payload is assembled by SQL views.', 'Seeded', 2),
    ('local', 'gateway', 'Event Gateway', 'The backend proxies operator intent to analytics and campaign events while Redpanda remains the only operational event ingress.', 'Connected', 3);

insert into analytics.platform_analytics_signal (
    environment,
    signal_id,
    title,
    description,
    action,
    event_kind,
    subject_type,
    subject_id,
    display_order
)
values
    ('local', 'nav-analytics', 'Open analytics view', 'Publishes an analytics event for navigation focus.', 'open-analytics-view', 'analytics', 'navigation', 'analytics', 1),
    ('local', 'create-brief', 'Create new brief', 'Creates an appdb-backed campaign row and emits analytics and campaign events to Redpanda.', 'create-brief', 'both', 'campaign', 'new-brief', 2),
    ('local', 'launch-test-send', 'Launch a test send', 'Updates the selected campaign while the emitted operational events stay in Redpanda topics.', 'launch-test-send', 'both', 'campaign', 'product-unveiling-zenith', 3);

insert into analytics.platform_campaign_analytics (
    environment,
    campaign_id,
    delivered_recipients,
    open_recipients,
    click_recipients,
    bounce_recipients,
    first_seen_at,
    last_seen_at,
    refreshed_at
)
values
    ('local', 'spring-capital-briefing', 4200, 1520, 390, 14, now() - interval '9 days', now() - interval '18 minutes', now() - interval '8 minutes'),
    ('local', 'monthly-security-digest', 0, 0, 0, 0, null, null, now() - interval '12 minutes'),
    ('local', 'product-unveiling-zenith', 0, 0, 0, 0, null, null, now() - interval '12 minutes'),
    ('local', 'quarterly-insight-report', 5300, 1140, 285, 9, now() - interval '15 days', now() - interval '1 day', now() - interval '1 day'),
    ('local', 'member-reengagement-arc', 1800, 360, 54, 6, now() - interval '7 days', now() - interval '4 minutes', now() - interval '4 minutes')
on conflict (environment, campaign_id) do update
set delivered_recipients = excluded.delivered_recipients,
    open_recipients = excluded.open_recipients,
    click_recipients = excluded.click_recipients,
    bounce_recipients = excluded.bounce_recipients,
    first_seen_at = excluded.first_seen_at,
    last_seen_at = excluded.last_seen_at,
    refreshed_at = excluded.refreshed_at;

insert into analytics.platform_engagement_analytics (
    environment,
    page_url,
    event_name,
    event_count,
    unique_users,
    first_seen_at,
    last_seen_at,
    refreshed_at,
    display_order
)
values
    ('local', '/dashboard', 'navigation.select', 218, 73, now() - interval '3 days', now() - interval '4 minutes', now() - interval '4 minutes', 1),
    ('local', '/campaigns', 'campaign-summary.inspect', 97, 31, now() - interval '2 days', now() - interval '16 minutes', now() - interval '16 minutes', 2),
    ('local', '/subscribers', 'subscriber-filters.select-filter', 164, 48, now() - interval '6 days', now() - interval '42 minutes', now() - interval '42 minutes', 3),
    ('local', '/analytics', 'telemetry-signal.open-analytics-view', 71, 22, now() - interval '4 days', now() - interval '83 minutes', now() - interval '83 minutes', 4)
on conflict (environment, page_url, event_name) do update
set event_count = excluded.event_count,
    unique_users = excluded.unique_users,
    first_seen_at = excluded.first_seen_at,
    last_seen_at = excluded.last_seen_at,
    refreshed_at = excluded.refreshed_at,
    display_order = excluded.display_order;

insert into analytics.platform_daily_performance (
    environment,
    bucket_date,
    event_count,
    refreshed_at
)
values
    ('local', current_date - interval '6 days', 3250, now() - interval '6 days'),
    ('local', current_date - interval '5 days', 5180, now() - interval '5 days'),
    ('local', current_date - interval '4 days', 4720, now() - interval '4 days'),
    ('local', current_date - interval '3 days', 3910, now() - interval '3 days'),
    ('local', current_date - interval '2 days', 5570, now() - interval '2 days'),
    ('local', current_date - interval '1 day', 6120, now() - interval '1 day'),
    ('local', current_date, 2980, now() - interval '4 minutes')
on conflict (environment, bucket_date) do update
set event_count = excluded.event_count,
    refreshed_at = excluded.refreshed_at;