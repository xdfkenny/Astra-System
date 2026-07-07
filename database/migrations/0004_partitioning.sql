-- 0004_partitioning.sql
-- Monthly range partitioning for audit_logs and analytics_events.
-- Creates default partitions plus one partition per month for the current
-- month and the next 12 months.

-- ---------------------------------------------------------------------------
-- UP
-- ---------------------------------------------------------------------------

-- Default partitions catch any rows outside the explicit monthly ranges.
CREATE TABLE IF NOT EXISTS audit_logs_default PARTITION OF audit_logs DEFAULT;
CREATE TABLE IF NOT EXISTS analytics_events_default PARTITION OF analytics_events DEFAULT;

-- Create monthly partitions for current month + next 12 months.
DO $$
DECLARE
  month_start DATE;
  month_end DATE;
  audit_partition TEXT;
  analytics_partition TEXT;
BEGIN
  month_start := DATE_TRUNC('month', CURRENT_DATE)::DATE;

  FOR i IN 0..12 LOOP
    month_end := (month_start + INTERVAL '1 month')::DATE;
    audit_partition := 'audit_logs_' || TO_CHAR(month_start, 'YYYY_MM');
    analytics_partition := 'analytics_events_' || TO_CHAR(month_start, 'YYYY_MM');

    EXECUTE format(
      'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L)',
      audit_partition,
      month_start,
      month_end
    );

    EXECUTE format(
      'CREATE TABLE IF NOT EXISTS %I PARTITION OF analytics_events FOR VALUES FROM (%L) TO (%L)',
      analytics_partition,
      month_start,
      month_end
    );

    month_start := month_end;
  END LOOP;
END $$;

-- Helper to create additional monthly partitions ahead of time.
-- Usage: SELECT create_future_audit_partition('2028-01-01'::date);
CREATE OR REPLACE FUNCTION create_future_audit_partition(partition_month DATE)
RETURNS TEXT AS $$
DECLARE
  partition_name TEXT;
  month_start DATE;
  month_end DATE;
BEGIN
  month_start := DATE_TRUNC('month', partition_month)::DATE;
  month_end := (month_start + INTERVAL '1 month')::DATE;
  partition_name := 'audit_logs_' || TO_CHAR(month_start, 'YYYY_MM');

  EXECUTE format(
    'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L)',
    partition_name,
    month_start,
    month_end
  );

  RETURN partition_name;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_future_analytics_partition(partition_month DATE)
RETURNS TEXT AS $$
DECLARE
  partition_name TEXT;
  month_start DATE;
  month_end DATE;
BEGIN
  month_start := DATE_TRUNC('month', partition_month)::DATE;
  month_end := (month_start + INTERVAL '1 month')::DATE;
  partition_name := 'analytics_events_' || TO_CHAR(month_start, 'YYYY_MM');

  EXECUTE format(
    'CREATE TABLE IF NOT EXISTS %I PARTITION OF analytics_events FOR VALUES FROM (%L) TO (%L)',
    partition_name,
    month_start,
    month_end
  );

  RETURN partition_name;
END;
$$ LANGUAGE plpgsql;

-- Convenience function to roll forward partitions by N months from today.
-- Usage: SELECT create_monthly_partitions(6);
CREATE OR REPLACE FUNCTION create_monthly_partitions(months_ahead INTEGER DEFAULT 1)
RETURNS TABLE (audit_partition TEXT, analytics_partition TEXT) AS $$
DECLARE
  target_date DATE;
BEGIN
  FOR i IN 1..months_ahead LOOP
    target_date := (DATE_TRUNC('month', CURRENT_DATE) + (i || ' months')::INTERVAL)::DATE;
    audit_partition := create_future_audit_partition(target_date);
    analytics_partition := create_future_analytics_partition(target_date);
    RETURN NEXT;
  END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- DOWN
-- ---------------------------------------------------------------------------

DROP FUNCTION IF EXISTS create_monthly_partitions(INTEGER);
DROP FUNCTION IF EXISTS create_future_analytics_partition(DATE);
DROP FUNCTION IF EXISTS create_future_audit_partition(DATE);

DO $$
DECLARE
  month_start DATE;
  audit_partition TEXT;
  analytics_partition TEXT;
BEGIN
  month_start := DATE_TRUNC('month', CURRENT_DATE)::DATE;

  FOR i IN 0..12 LOOP
    audit_partition := 'audit_logs_' || TO_CHAR(month_start, 'YYYY_MM');
    analytics_partition := 'analytics_events_' || TO_CHAR(month_start, 'YYYY_MM');

    EXECUTE format('DROP TABLE IF EXISTS %I', audit_partition);
    EXECUTE format('DROP TABLE IF EXISTS %I', analytics_partition);

    month_start := (month_start + INTERVAL '1 month')::DATE;
  END LOOP;
END $$;

DROP TABLE IF EXISTS audit_logs_default;
DROP TABLE IF EXISTS analytics_events_default;
