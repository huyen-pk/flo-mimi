with exported_statements as (
    select 1 as section_order,
           format(
               'insert into analytics.platform_environment (environment, brand_name, brand_tagline, hero_title, hero_accent, hero_note, search_placeholder, session_label, session_node, session_status, campaigns_headline, campaigns_description, subscribers_headline, subscribers_description, analytics_headline, analytics_description, security_title, security_status, security_description, billing_eyebrow, billing_title, billing_date, billing_action) values (%L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L);',
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
           ) as sql_text
    from analytics.platform_environment

    union all

    select 2,
           format(
               'insert into analytics.platform_filter (environment, filter_label, display_order) values (%L, %L, %s);',
               environment,
               filter_label,
               display_order
           )
    from analytics.platform_filter

    union all

    select 3,
           format(
               'insert into analytics.platform_dashboard_activity (environment, activity_id, title, description, occurred_at, tone, display_order) values (%L, %L, %L, %L, %L, %L, %s);',
               environment,
               activity_id,
               title,
               description,
               occurred_at,
               tone,
               display_order
           )
    from analytics.platform_dashboard_activity

    union all

    select 4,
           format(
               'insert into analytics.platform_growth_segment (environment, segment_id, label, percentage, display_order) values (%L, %L, %L, %s, %s);',
               environment,
               segment_id,
               label,
               percentage,
               display_order
           )
    from analytics.platform_growth_segment

    union all

    select 5,
           format(
               'insert into analytics.platform_campaign (environment, campaign_id, title, summary, status, audience_count, audience_label, action_label, tone, display_order) values (%L, %L, %L, %L, %L, %s, %L, %L, %L, %s);',
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
    from analytics.platform_campaign

    union all

    select 6,
           format(
               'insert into analytics.platform_subscriber (environment, subscriber_id, name, email, security_status, created_at, last_interaction_at, tone, is_featured, display_order) values (%L, %L, %L, %L, %L, %L, %L, %L, %L, %s);',
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
    from analytics.platform_subscriber

    union all

    select 7,
           format(
               'insert into analytics.platform_subscriber_segment (environment, subscriber_id, segment_label, display_order) values (%L, %L, %L, %s);',
               environment,
               subscriber_id,
               segment_label,
               display_order
           )
    from analytics.platform_subscriber_segment

    union all

    select 8,
           format(
               'insert into analytics.platform_filter_assignment (environment, subscriber_id, filter_label) values (%L, %L, %L);',
               environment,
               subscriber_id,
               filter_label
           )
    from analytics.platform_filter_assignment

    union all

    select 9,
           format(
               'insert into analytics.platform_analytics_pipeline (environment, pipeline_id, title, description, status, display_order) values (%L, %L, %L, %L, %L, %s);',
               environment,
               pipeline_id,
               title,
               description,
               status,
               display_order
           )
    from analytics.platform_analytics_pipeline

    union all

    select 10,
           format(
               'insert into analytics.platform_analytics_signal (environment, signal_id, title, description, action, event_kind, subject_type, subject_id, display_order) values (%L, %L, %L, %L, %L, %L, %L, %L, %s);',
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
    from analytics.platform_analytics_signal

    union all

    select 11,
           format(
               'insert into analytics.platform_campaign_analytics (environment, campaign_id, delivered_recipients, open_recipients, click_recipients, bounce_recipients, first_seen_at, last_seen_at, refreshed_at) values (%L, %L, %s, %s, %s, %s, %L, %L, %L) on conflict (environment, campaign_id) do update set delivered_recipients = excluded.delivered_recipients, open_recipients = excluded.open_recipients, click_recipients = excluded.click_recipients, bounce_recipients = excluded.bounce_recipients, first_seen_at = excluded.first_seen_at, last_seen_at = excluded.last_seen_at, refreshed_at = excluded.refreshed_at;',
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
    from analytics.platform_campaign_analytics

    union all

    select 12,
           format(
               'insert into analytics.platform_engagement_analytics (environment, page_url, event_name, event_count, unique_users, first_seen_at, last_seen_at, refreshed_at, display_order) values (%L, %L, %L, %s, %s, %L, %L, %L, %s) on conflict (environment, page_url, event_name) do update set event_count = excluded.event_count, unique_users = excluded.unique_users, first_seen_at = excluded.first_seen_at, last_seen_at = excluded.last_seen_at, refreshed_at = excluded.refreshed_at, display_order = excluded.display_order;',
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
    from analytics.platform_engagement_analytics

    union all

    select 13,
           format(
               'insert into analytics.platform_daily_performance (environment, bucket_date, event_count, refreshed_at) values (%L, %L, %s, %L) on conflict (environment, bucket_date) do update set event_count = excluded.event_count, refreshed_at = excluded.refreshed_at;',
               environment,
               bucket_date,
               event_count,
               refreshed_at
           )
    from analytics.platform_daily_performance
)
select sql_text
from exported_statements
order by section_order, sql_text;