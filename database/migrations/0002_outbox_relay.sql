-- 0002_outbox_relay.sql
-- Transactional outbox table and relay helper functions.
-- The outbox_events table is also created in 0001_init.sql; the definition
-- below is idempotent so this migration can be applied independently.

-- ---------------------------------------------------------------------------
-- UP
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS outbox_events (
  event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  aggregate_type VARCHAR(64) NOT NULL,
  aggregate_id UUID NOT NULL,
  event_type VARCHAR(128) NOT NULL,
  payload JSONB NOT NULL,
  occurred_at_ms BIGINT NOT NULL,
  published BOOLEAN NOT NULL DEFAULT FALSE,
  published_at_ms BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
  ON outbox_events(published, occurred_at_ms)
  WHERE published = FALSE;

CREATE INDEX IF NOT EXISTS idx_outbox_aggregate
  ON outbox_events(aggregate_type, aggregate_id, occurred_at_ms);

-- Relay helper: claim and return the next batch of unpublished events.
-- Callers should publish the returned rows and rely on the update already
-- marking them published. SKIP LOCKED prevents relay workers from stepping
-- on each other.
CREATE OR REPLACE FUNCTION publish_outbox_batch(batch_size INTEGER DEFAULT 100)
RETURNS TABLE (
  event_id UUID,
  aggregate_type VARCHAR(64),
  aggregate_id UUID,
  event_type VARCHAR(128),
  payload JSONB,
  occurred_at_ms BIGINT
) AS $$
BEGIN
  RETURN QUERY
  UPDATE outbox_events
  SET published = TRUE,
      published_at_ms = (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
  WHERE outbox_events.event_id IN (
    SELECT outbox_events.event_id
    FROM outbox_events
    WHERE outbox_events.published = FALSE
    ORDER BY outbox_events.occurred_at_ms ASC
    LIMIT batch_size
    FOR UPDATE SKIP LOCKED
  )
  RETURNING outbox_events.event_id,
            outbox_events.aggregate_type,
            outbox_events.aggregate_id,
            outbox_events.event_type,
            outbox_events.payload,
            outbox_events.occurred_at_ms;
END;
$$ LANGUAGE plpgsql;

-- Relay helper: remove events that were published before the supplied
-- timestamp (in milliseconds since epoch). Returns the number of rows removed.
CREATE OR REPLACE FUNCTION prune_published_outbox(older_than_ms BIGINT)
RETURNS INTEGER AS $$
DECLARE
  deleted_count INTEGER;
BEGIN
  DELETE FROM outbox_events
  WHERE published = TRUE
    AND published_at_ms < older_than_ms;
  GET DIAGNOSTICS deleted_count = ROW_COUNT;
  RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- DOWN
-- ---------------------------------------------------------------------------

DROP FUNCTION IF EXISTS prune_published_outbox(BIGINT);
DROP FUNCTION IF EXISTS publish_outbox_batch(INTEGER);
DROP TABLE IF EXISTS outbox_events CASCADE;
