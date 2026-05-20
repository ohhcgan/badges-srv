CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- gen_random_uuid

-- users table
CREATE TABLE IF NOT EXISTS users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id       BIGINT      NOT NULL,
    github_login    TEXT        NOT NULL,
    name            TEXT        NOT NULL DEFAULT '',
    email           TEXT        NOT NULL DEFAULT '',
    avatar_url      TEXT        NOT NULL DEFAULT '',
    encrypted_token TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_users_github_id UNIQUE (github_id)
);
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users (github_id);

-- monthly_stats  table
CREATE TABLE IF NOT EXISTS monthly_stats (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stat_month                  DATE        NOT NULL,
    total_commits               INT         NOT NULL DEFAULT 0,
    repos_created               INT         NOT NULL DEFAULT 0,
    open_source_contributions   INT         NOT NULL DEFAULT 0,
    commit_pct_change           NUMERIC(10, 4),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_monthly_stats_user_month UNIQUE (user_id, stat_month)
);
CREATE INDEX IF NOT EXISTS idx_monthly_stats_user_month
    ON monthly_stats (user_id, stat_month DESC);

-- email_logs table
CREATE TABLE IF NOT EXISTS email_logs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stat_month DATE        NOT NULL,
    status     TEXT        NOT NULL CHECK (status IN ('sent', 'failed')),
    error_msg  TEXT,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
