CREATE TABLE agents (
  id             TEXT      PRIMARY KEY,
  namespace      TEXT      NOT NULL,
  metadata       JSONB     NOT NULL DEFAULT '{}',
  secret_hash    BYTEA     NOT NULL,
  last_seen      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status         TEXT      NOT NULL DEFAULT 'active',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agents_status ON agents(status);

CREATE INDEX idx_agents_last_seen ON agents(last_seen);

CREATE INDEX idx_agents_namespace ON agents(namespace);

