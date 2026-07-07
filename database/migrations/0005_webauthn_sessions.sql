-- 0005_webauthn_sessions.sql
-- Session state for the WebAuthn verification service.

-- ---------------------------------------------------------------------------
-- UP
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS webauthn_sessions (
  session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_id UUID NOT NULL,
  actor_type VARCHAR(16) NOT NULL,
  challenge VARCHAR(255) NOT NULL,
  store_id UUID,
  kiosk_id UUID,
  tenant_id UUID,
  reason VARCHAR(255),
  relying_party_id VARCHAR(255),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_actor_challenge
  ON webauthn_sessions(actor_id, actor_type, challenge);

CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires
  ON webauthn_sessions(expires_at);

-- ---------------------------------------------------------------------------
-- DOWN
-- ---------------------------------------------------------------------------

DROP TABLE IF EXISTS webauthn_sessions CASCADE;
