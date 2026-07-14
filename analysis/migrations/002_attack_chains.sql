-- Attack chains: multi-step successful campaigns keyed by chain_id in battle
-- event labels. Steps also live in battle_events; these tables are the
-- queryable chain index for the dashboard.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS attack_chains (
    id                   UUID PRIMARY KEY,
    started_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    status               TEXT        NOT NULL DEFAULT 'active',
    -- active | landed | contained | abandoned
    strategy             TEXT        NOT NULL DEFAULT '',
    landed_steps         INT         NOT NULL DEFAULT 0,
    technique_path       TEXT[]      NOT NULL DEFAULT '{}',
    remediation_summary  TEXT        NOT NULL DEFAULT '',
    labels               JSONB       NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_attack_chains_status_updated
    ON attack_chains (status, updated_at DESC);

CREATE TABLE IF NOT EXISTS attack_chain_steps (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id         UUID        NOT NULL REFERENCES attack_chains(id) ON DELETE CASCADE,
    step_index       INT         NOT NULL,
    session_id       TEXT        NOT NULL DEFAULT '',
    event_id         UUID,
    technique        TEXT        NOT NULL DEFAULT '',
    variant          TEXT        NOT NULL DEFAULT '',
    outcome          TEXT        NOT NULL DEFAULT '',
    mutation_source  TEXT        NOT NULL DEFAULT 'deterministic',
    strategy_reason  TEXT        NOT NULL DEFAULT '',
    payload_preview  TEXT        NOT NULL DEFAULT '',
    ts               TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chain_id, step_index)
);

CREATE INDEX IF NOT EXISTS idx_attack_chain_steps_chain
    ON attack_chain_steps (chain_id, step_index);
