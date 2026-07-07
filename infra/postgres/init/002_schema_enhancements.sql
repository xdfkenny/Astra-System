-- Astra-Service PostgreSQL Schema Enhancements
-- Version: 2.0.0
-- Description: Adds tenant/locations/lanes hierarchy, RBAC, refunds,
--              inventory ledger, partitioned audit log, and hardened RLS.

-- ───────────────────────────────────────────────────────────
-- Tenant / Location / Lane Hierarchy
-- ───────────────────────────────────────────────────────────
CREATE TABLE tenants (
    tenant_id           UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    slug                VARCHAR(64) NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL,
    billing_email       VARCHAR(255) NOT NULL,
    plan                VARCHAR(16) NOT NULL DEFAULT 'standard' CHECK (plan IN ('standard', 'enterprise')),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;

CREATE TABLE locations (
    location_id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id),
    slug                VARCHAR(64) NOT NULL,
    name                VARCHAR(255) NOT NULL,
    address             TEXT,
    timezone            VARCHAR(64) NOT NULL DEFAULT 'UTC',
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    tax_rate            DECIMAL(5,4) NOT NULL DEFAULT 0.0000 CHECK (tax_rate >= 0 AND tax_rate <= 1),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_locations_tenant ON locations(tenant_id, deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE lanes (
    lane_id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    location_id         UUID NOT NULL REFERENCES locations(location_id),
    display_name        VARCHAR(64) NOT NULL,
    lane_number         INT NOT NULL,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    UNIQUE (location_id, lane_number)
);

CREATE INDEX idx_lanes_location ON lanes(location_id, deleted_at) WHERE deleted_at IS NULL;

-- Backfill stores table with tenant/location linkage.
ALTER TABLE stores
    ADD COLUMN tenant_id UUID REFERENCES tenants(tenant_id),
    ADD COLUMN location_id UUID REFERENCES locations(location_id);

CREATE INDEX idx_stores_tenant ON stores(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_stores_location ON stores(location_id, deleted_at) WHERE deleted_at IS NULL;

-- Kiosks now belong to lanes.
ALTER TABLE kiosks
    ADD COLUMN lane_id UUID REFERENCES lanes(lane_id),
    ADD COLUMN tenant_id UUID REFERENCES tenants(tenant_id);

CREATE INDEX idx_kiosks_lane ON kiosks(lane_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_kiosks_tenant ON kiosks(tenant_id, deleted_at) WHERE deleted_at IS NULL;

-- ───────────────────────────────────────────────────────────
-- RBAC
-- ───────────────────────────────────────────────────────────
CREATE TABLE roles (
    role_id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id),
    name                VARCHAR(64) NOT NULL,
    description         TEXT,
    is_system           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);

CREATE TABLE permissions (
    permission_id       UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    resource            VARCHAR(64) NOT NULL, -- 'order', 'inventory', etc.
    action              VARCHAR(64) NOT NULL, -- 'read', 'refund', etc.
    description         TEXT,
    UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
    role_id             UUID NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
    permission_id       UUID NOT NULL REFERENCES permissions(permission_id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    user_id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id),
    email               VARCHAR(255) NOT NULL,
    name                VARCHAR(128) NOT NULL,
    role_id             UUID NOT NULL REFERENCES roles(role_id),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    webauthn_credential_id VARCHAR(255),
    webauthn_public_key   BYTEA,
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role_id);

-- ───────────────────────────────────────────────────────────
-- Refunds
-- ───────────────────────────────────────────────────────────
CREATE TABLE refunds (
    refund_id           UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    payment_id          UUID NOT NULL REFERENCES payments(payment_id),
    order_id            UUID NOT NULL REFERENCES orders(order_id),
    kiosk_id            UUID NOT NULL REFERENCES kiosks(kiosk_id),
    amount_cents        INT NOT NULL CHECK (amount_cents > 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    reason              VARCHAR(255) NOT NULL,
    status              VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed')),
    verifone_reference  VARCHAR(255),
    processed_by        UUID REFERENCES employees(employee_id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refunds_payment ON refunds(payment_id);
CREATE INDEX idx_refunds_order ON refunds(order_id, created_at DESC);

-- ───────────────────────────────────────────────────────────
-- Inventory Ledger (immutable, never UPDATE-in-place)
-- Replaces inventory_movements as the source of truth for stock history.
-- ───────────────────────────────────────────────────────────
CREATE TABLE inventory_transactions (
    transaction_id      UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    item_id             UUID NOT NULL REFERENCES items(item_id),
    transaction_type    VARCHAR(16) NOT NULL CHECK (transaction_type IN ('sale', 'restock', 'adjustment', 'reserved', 'released', 'waste', 'return')),
    quantity_delta      INT NOT NULL,
    running_balance     INT NOT NULL,
    reference_id        UUID, -- order_id, adjustment_id, etc.
    reference_type      VARCHAR(32), -- 'order', 'adjustment', 'reservation'
    kiosk_id            UUID REFERENCES kiosks(kiosk_id),
    employee_id         UUID REFERENCES employees(employee_id),
    notes               VARCHAR(500),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_transactions_store_item ON inventory_transactions(store_id, item_id, created_at DESC);
CREATE INDEX idx_inventory_transactions_reference ON inventory_transactions(reference_type, reference_id);

-- Drop the old movements table in favor of the ledger.
DROP TABLE IF EXISTS inventory_movements;

-- Update inventory trigger to write to ledger instead.
CREATE OR REPLACE FUNCTION log_inventory_movement()
RETURNS TRIGGER AS $$
DECLARE
    delta INT;
BEGIN
    delta := NEW.quantity_available - COALESCE(OLD.quantity_available, 0);
    IF delta != 0 THEN
        INSERT INTO inventory_transactions (
            store_id, item_id, transaction_type, quantity_delta, running_balance,
            reference_type, kiosk_id, notes, created_at
        ) VALUES (
            NEW.store_id, NEW.item_id,
            CASE WHEN delta > 0 THEN 'restock' ELSE 'adjustment' END,
            delta, NEW.quantity_available,
            'adjustment', NULL, 'automatic: inventory update', NOW()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ───────────────────────────────────────────────────────────
-- Event Store (append-only domain events)
-- ───────────────────────────────────────────────────────────
CREATE TABLE event_store (
    event_id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    event_schema        VARCHAR(64) NOT NULL, -- e.g. 'astra.order.created.v1'
    aggregate_type      VARCHAR(64) NOT NULL,
    aggregate_id        UUID NOT NULL,
    sequence_number     BIGINT NOT NULL,
    payload             JSONB NOT NULL,
    metadata            JSONB NOT NULL DEFAULT '{}',
    occurred_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (aggregate_type, aggregate_id, sequence_number)
);

CREATE INDEX idx_event_store_aggregate ON event_store(aggregate_type, aggregate_id, sequence_number);
CREATE INDEX idx_event_store_occurred ON event_store(occurred_at);

-- ───────────────────────────────────────────────────────────
-- Partitioned Audit Log
-- ───────────────────────────────────────────────────────────
-- Migrate existing audit_log into partitioned table.
CREATE TABLE audit_logs (
    audit_id            BIGSERIAL,
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    tenant_id           UUID REFERENCES tenants(tenant_id),
    lane_id             UUID REFERENCES lanes(lane_id),
    kiosk_id            UUID REFERENCES kiosks(kiosk_id),
    employee_id         UUID REFERENCES employees(employee_id),
    user_id             UUID REFERENCES users(user_id),
    event_type          VARCHAR(32) NOT NULL,
    entity_type         VARCHAR(32) NOT NULL,
    entity_id           UUID NOT NULL,
    payload_json        JSONB NOT NULL,
    previous_hash       VARCHAR(64) NOT NULL,
    current_hash        VARCHAR(64) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (audit_id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX idx_audit_logs_store ON audit_logs(store_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at DESC);
CREATE INDEX idx_audit_logs_event ON audit_logs(event_type, created_at DESC);

-- Initial partitions for 2025-2026.
DO $$
DECLARE
    y INT;
    m INT;
    start_date DATE;
    end_date DATE;
    part_name TEXT;
BEGIN
    FOR y IN 2025..2026 LOOP
        FOR m IN 1..12 LOOP
            start_date := make_date(y, m, 1);
            end_date := start_date + INTERVAL '1 month';
            part_name := format('audit_logs_%s_%s', y, LPAD(m::TEXT, 2, '0'));
            EXECUTE format(
                'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L);',
                part_name, start_date, end_date
            );
        END LOOP;
    END LOOP;
END $$;

-- ───────────────────────────────────────────────────────────
-- Updated At Triggers for New Tables
-- ───────────────────────────────────────────────────────────
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_locations_updated_at BEFORE UPDATE ON locations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_lanes_updated_at BEFORE UPDATE ON lanes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_refunds_updated_at BEFORE UPDATE ON refunds FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ───────────────────────────────────────────────────────────
-- Hardened RLS Policies
-- ───────────────────────────────────────────────────────────
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE lanes ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE refunds ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_store ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- Application sets session variable 'app.current_tenant_id' on each connection.
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_tenant_id', TRUE), '')::UUID;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE POLICY tenant_isolation_tenants ON tenants
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_locations ON locations
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_lanes ON lanes
    USING (location_id IN (
        SELECT location_id FROM locations WHERE tenant_id = current_tenant_id()
    ) OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_roles ON roles
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_stores ON stores
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_kiosks ON kiosks
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_orders ON orders
    USING (store_id IN (
        SELECT store_id FROM stores WHERE tenant_id = current_tenant_id()
    ) OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_payments ON payments
    USING (order_id IN (
        SELECT order_id FROM orders WHERE store_id IN (
            SELECT store_id FROM stores WHERE tenant_id = current_tenant_id()
        )
    ) OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

-- ───────────────────────────────────────────────────────────
-- Seed data: tenant, location, lanes, RBAC permissions
-- ───────────────────────────────────────────────────────────
INSERT INTO tenants (tenant_id, slug, name, billing_email) VALUES
    ('11111111-1111-1111-1111-111111111111', 'astra-demo', 'Astra Demo Tenant', 'billing@astra-demo.internal');

INSERT INTO locations (location_id, tenant_id, slug, name, address, timezone, currency, tax_rate) VALUES
    ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'miami-brickell', 'Astra Miami Brickell', '1000 Brickell Ave, Miami, FL 33131', 'America/New_York', 'USD', 0.0700);

UPDATE stores SET tenant_id = '11111111-1111-1111-1111-111111111111', location_id = '22222222-2222-2222-2222-222222222222'
    WHERE store_id = '550e8400-e29b-41d4-a716-446655440000';

INSERT INTO lanes (lane_id, location_id, display_name, lane_number) VALUES
    ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'Lane 1', 1),
    ('44444444-4444-4444-4444-444444444444', '22222222-2222-2222-2222-222222222222', 'Lane 2', 2);

UPDATE kiosks SET tenant_id = '11111111-1111-1111-1111-111111111111', lane_id = '33333333-3333-3333-3333-333333333333'
    WHERE hardware_id = 'HW-KIOSK-001';
UPDATE kiosks SET tenant_id = '11111111-1111-1111-1111-111111111111', lane_id = '44444444-4444-4444-4444-444444444444'
    WHERE hardware_id = 'HW-KIOSK-002';

INSERT INTO permissions (resource, action, description) VALUES
    ('order', 'read', 'View orders'),
    ('order', 'refund', 'Process refunds'),
    ('order', 'cancel', 'Cancel orders'),
    ('inventory', 'read', 'View inventory'),
    ('inventory', 'adjust', 'Adjust inventory counts'),
    ('employee', 'manage', 'Manage employees'),
    ('report', 'read', 'View reports'),
    ('kiosk', 'manage', 'Manage kiosk settings');

INSERT INTO roles (role_id, tenant_id, name, description, is_system) VALUES
    ('55555555-5555-5555-5555-555555555555', '11111111-1111-1111-1111-111111111111', 'cashier', 'Standard cashier', TRUE),
    ('66666666-6666-6666-6666-666666666666', '11111111-1111-1111-1111-111111111111', 'supervisor', 'Can refund and adjust inventory', TRUE),
    ('77777777-7777-7777-7777-777777777777', '11111111-1111-1111-1111-111111111111', 'manager', 'Full location access', TRUE);

INSERT INTO role_permissions (role_id, permission_id)
SELECT '55555555-5555-5555-5555-555555555555', permission_id FROM permissions WHERE (resource, action) IN (('order', 'read'));

INSERT INTO role_permissions (role_id, permission_id)
SELECT '66666666-6666-6666-6666-666666666666', permission_id FROM permissions WHERE (resource, action) IN (('order', 'read'), ('order', 'refund'), ('inventory', 'read'), ('inventory', 'adjust'));

INSERT INTO role_permissions (role_id, permission_id)
SELECT '77777777-7777-7777-7777-777777777777', permission_id FROM permissions;
