do $$
declare
    campaign_table_needs_reset boolean;
    engagement_table_needs_reset boolean;
begin
    select exists (
        select 1
        from information_schema.tables as table_info
        where table_info.table_schema = 'analytics'
          and table_info.table_name = 'platform_campaign_analytics'
          and (
              not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'environment'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'campaign_id'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'delivered_recipients'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'open_recipients'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'click_recipients'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'bounce_recipients'
              )
          )
    ) into campaign_table_needs_reset;

    select exists (
        select 1
        from information_schema.tables as table_info
        where table_info.table_schema = 'analytics'
          and table_info.table_name = 'platform_engagement_analytics'
          and (
              not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'environment'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'page_url'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'event_name'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'event_count'
              )
              or not exists (
                  select 1
                  from information_schema.columns as column_info
                  where column_info.table_schema = table_info.table_schema
                    and column_info.table_name = table_info.table_name
                    and column_info.column_name = 'unique_users'
              )
          )
    ) into engagement_table_needs_reset;

    if not campaign_table_needs_reset and not engagement_table_needs_reset then
        return;
    end if;

    drop view if exists analytics.platform_bootstrap;
    drop view if exists analytics.platform_dashboard_bars_view;
    drop view if exists analytics.platform_campaign_summary_view;
    drop view if exists analytics.platform_dashboard_metrics_view;
    drop view if exists analytics.platform_campaign_live_metrics_view;

    drop table if exists analytics.platform_engagement_analytics;
    drop table if exists analytics.platform_campaign_analytics;
end
$$;