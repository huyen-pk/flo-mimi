create schema if not exists raw;
create schema if not exists analytics;

create table if not exists raw.raw_third_party_records (
    record_id text primary key,
    provider text not null,
    account_id text,
    collected_at timestamptz not null,
    payload jsonb not null default '{}'::jsonb
);

create index if not exists idx_raw_third_party_provider on raw.raw_third_party_records (provider);
create index if not exists idx_raw_third_party_collected_at on raw.raw_third_party_records (collected_at);