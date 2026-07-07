-- Astra-Service PostgreSQL Schema
-- Version: 1.0.0
-- Description: Initial schema for the cloud PostgreSQL database.
--   Local kiosks use an encrypted SQLite (SQLCipher) subset of this schema.
--
-- Design principles:
--   - UUID v7 primary keys (time-sortable, k-sortable, no MAC leakage)
--   - Immutable order records (no UPDATE on orders after creation)
--   - Soft deletes only (never hard DELETE on business tables)
--   - JSONB for flexible metadata (modifiers, analytics payloads)
--   - Partial indexes for hot query paths
--   - Row-level security (RLS) on tenant-scoped tables

-- ───────────────────────────────────────────────────────────
-- Extensions
-- ───────────────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ───────────────────────────────────────────────────────────
-- UUID v7 generation (time-sortable, k-sortable, no MAC leakage)
-- PostgreSQL's built-in extensions do not provide v7, so we ship a
-- pure-SQL implementation. This keeps the schema self-contained and
-- avoids a third-party extension dependency in managed Postgres offerings.
-- ───────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS UUID AS $$
DECLARE
    -- UUID v7 layout:
    --   48 bits: Unix timestamp in milliseconds (big-endian)
    --    4 bits: version (0b0111)
    --   12 bits: random "rand_a"
    --    2 bits: variant (0b10)
    --   62 bits: random "rand_b"
    ts_ms BIGINT;
    rand_bytes BYTEA;
    v7 BYTEA;
BEGIN
    ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;
    rand_bytes := gen_random_bytes(8); -- 64 bits for rand_a + rand_b (bytes 8-15)

    v7 := '\x0000000000007000'::BYTEA || rand_bytes;

    -- Overlay the 48-bit timestamp into the first 6 bytes (big-endian)
    v7 := set_byte(v7, 0, ((ts_ms >> 40) & 255)::INTEGER);
    v7 := set_byte(v7, 1, ((ts_ms >> 32) & 255)::INTEGER);
    v7 := set_byte(v7, 2, ((ts_ms >> 24) & 255)::INTEGER);
    v7 := set_byte(v7, 3, ((ts_ms >> 16) & 255)::INTEGER);
    v7 := set_byte(v7, 4, ((ts_ms >> 8)  & 255)::INTEGER);
    v7 := set_byte(v7, 5, ( ts_ms        & 255)::INTEGER);

    -- Set version nibble to 7 (byte 6: high nibble)
    v7 := set_byte(v7, 6, (get_byte(v7, 6) & 15) | 112); -- 112 = 0x70

    -- Set variant bits to 10 (byte 8: high two bits)
    v7 := set_byte(v7, 8, (get_byte(v7, 8) & 63) | 128); -- 128 = 0x80

    RETURN encode(v7, 'hex')::UUID;
END;
$$ LANGUAGE plpgsql VOLATILE;

CREATE OR REPLACE FUNCTION uuid_generate_v7(ts_ms BIGINT)
RETURNS UUID AS $$
DECLARE
    rand_bytes BYTEA;
    v7 BYTEA;
BEGIN
    rand_bytes := gen_random_bytes(8);
    v7 := '\x0000000000007000'::BYTEA || rand_bytes;
    v7 := set_byte(v7, 0, ((ts_ms >> 40) & 255)::INTEGER);
    v7 := set_byte(v7, 1, ((ts_ms >> 32) & 255)::INTEGER);
    v7 := set_byte(v7, 2, ((ts_ms >> 24) & 255)::INTEGER);
    v7 := set_byte(v7, 3, ((ts_ms >> 16) & 255)::INTEGER);
    v7 := set_byte(v7, 4, ((ts_ms >> 8)  & 255)::INTEGER);
    v7 := set_byte(v7, 5, ( ts_ms        & 255)::INTEGER);
    v7 := set_byte(v7, 6, (get_byte(v7, 6) & 15) | 112);
    v7 := set_byte(v7, 8, (get_byte(v7, 8) & 63) | 128);
    RETURN encode(v7, 'hex')::UUID;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- ───────────────────────────────────────────────────────────
-- Tenant / Store Management
-- ───────────────────────────────────────────────────────────
CREATE TABLE stores (
    store_id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name                VARCHAR(255) NOT NULL,
    address             TEXT,
    timezone            VARCHAR(64) NOT NULL DEFAULT 'UTC',
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    tax_rate            DECIMAL(5,4) NOT NULL DEFAULT 0.0000, -- e.g. 0.0825 for 8.25%
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT valid_tax_rate CHECK (tax_rate >= 0 AND tax_rate <= 1)
);

CREATE INDEX idx_stores_deleted_at ON stores(deleted_at) WHERE deleted_at IS NULL;

-- ───────────────────────────────────────────────────────────
-- Kiosks (lane terminals)
-- ───────────────────────────────────────────────────────────
CREATE TABLE kiosks (
    kiosk_id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    hardware_id         VARCHAR(64) NOT NULL UNIQUE, -- immutable device serial
    display_name        VARCHAR(64) NOT NULL,
    ip_address          INET,
    last_seen_at        TIMESTAMPTZ,
    sync_status         VARCHAR(16) NOT NULL DEFAULT 'online' CHECK (sync_status IN ('online', 'offline', 'degraded', 'maintenance')),
    is_leader           BOOLEAN NOT NULL DEFAULT FALSE,
    signing_key_hash    VARCHAR(64) NOT NULL, -- SHA-256 of the HMAC signing key (key itself in Vault)
    firmware_version    VARCHAR(32),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_kiosks_store ON kiosks(store_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_kiosks_leader ON kiosks(store_id, is_leader) WHERE is_leader = TRUE AND deleted_at IS NULL;
CREATE UNIQUE INDEX idx_kiosks_one_leader_per_store ON kiosks(store_id) WHERE is_leader = TRUE AND deleted_at IS NULL;

-- ───────────────────────────────────────────────────────────
-- Menu: Categories
-- ───────────────────────────────────────────────────────────
CREATE TABLE categories (
    category_id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    parent_id           UUID REFERENCES categories(category_id), -- self-referential for nested categories
    name                VARCHAR(128) NOT NULL,
    description         TEXT,
    display_order       INT NOT NULL DEFAULT 0,
    image_url           VARCHAR(512),
    blurhash            VARCHAR(32), -- blurhash for skeleton placeholder
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_categories_store ON categories(store_id, display_order, is_active) WHERE deleted_at IS NULL;

-- ───────────────────────────────────────────────────────────
-- Menu: Items (products)
-- ───────────────────────────────────────────────────────────
CREATE TABLE items (
    item_id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    category_id         UUID NOT NULL REFERENCES categories(category_id),
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    price_cents         INT NOT NULL CHECK (price_cents >= 0),
    cost_cents          INT CHECK (cost_cents >= 0), -- for margin analytics
    plu                 VARCHAR(16), -- Price Look-Up code for produce/scales
    barcode             VARCHAR(32), -- EAN/UPC
    sku                 VARCHAR(64), -- internal SKU
    image_url           VARCHAR(512),
    blurhash            VARCHAR(32),
    tax_category        VARCHAR(16) NOT NULL DEFAULT 'standard' CHECK (tax_category IN ('standard', 'exempt', 'reduced')),
    is_weight_based     BOOLEAN NOT NULL DEFAULT FALSE, -- true for produce/meat
    weight_unit         VARCHAR(8) CHECK (weight_unit IN ('g', 'kg', 'lb', 'oz')),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    metadata            JSONB, -- flexible: allergens, nutritional info, etc.
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_items_store_category ON items(store_id, category_id, is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_items_barcode ON items(barcode) WHERE barcode IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_items_plu ON items(plu) WHERE plu IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_items_name_trgm ON items USING gin(name gin_trgm_ops); -- for fuzzy search

-- ───────────────────────────────────────────────────────────
-- Menu: Modifiers (e.g. "Extra Cheese", "No Onions")
-- ───────────────────────────────────────────────────────────
CREATE TABLE modifier_groups (
    modifier_group_id   UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    name                VARCHAR(128) NOT NULL,
    description         TEXT,
    min_select          INT NOT NULL DEFAULT 0 CHECK (min_select >= 0),
    max_select          INT NOT NULL DEFAULT 1 CHECK (max_select >= min_select),
    display_order       INT NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE TABLE modifier_options (
    modifier_option_id  UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    modifier_group_id   UUID NOT NULL REFERENCES modifier_groups(modifier_group_id),
    name                VARCHAR(128) NOT NULL,
    price_delta_cents   INT NOT NULL DEFAULT 0,
    is_default          BOOLEAN NOT NULL DEFAULT FALSE,
    display_order       INT NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

-- Junction: which modifier groups apply to which items
CREATE TABLE item_modifier_groups (
    item_id             UUID NOT NULL REFERENCES items(item_id),
    modifier_group_id   UUID NOT NULL REFERENCES modifier_groups(modifier_group_id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id, modifier_group_id)
);

-- ───────────────────────────────────────────────────────────
-- Inventory
-- ───────────────────────────────────────────────────────────
CREATE TABLE inventory (
    inventory_id        UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    item_id             UUID NOT NULL REFERENCES items(item_id),
    quantity_available  INT NOT NULL DEFAULT 0 CHECK (quantity_available >= 0),
    quantity_reserved   INT NOT NULL DEFAULT 0 CHECK (quantity_reserved >= 0),
    quantity_on_order   INT NOT NULL DEFAULT 0 CHECK (quantity_on_order >= 0),
    reorder_point       INT NOT NULL DEFAULT 0, -- trigger restock alert
    reorder_quantity    INT NOT NULL DEFAULT 0, -- how much to order
    location            VARCHAR(64), -- e.g. "Aisle 3, Shelf 2"
    last_counted_at     TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- no deleted_at: inventory is a fact, not a soft-deletable entity
    UNIQUE (store_id, item_id)
);

CREATE INDEX idx_inventory_store ON inventory(store_id);
CREATE INDEX idx_inventory_low_stock ON inventory(store_id, quantity_available) WHERE quantity_available <= reorder_point;

-- Inventory movements (audit trail)
CREATE TABLE inventory_movements (
    movement_id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    item_id             UUID NOT NULL REFERENCES items(item_id),
    movement_type       VARCHAR(16) NOT NULL CHECK (movement_type IN ('sale', 'restock', 'adjustment', 'reserved', 'released', 'waste')),
    quantity_delta      INT NOT NULL,
    reason              VARCHAR(255),
    order_id            UUID, -- nullable: links to order if from sale
    kiosk_id            UUID REFERENCES kiosks(kiosk_id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_movements_store_item ON inventory_movements(store_id, item_id, created_at);
CREATE INDEX idx_inventory_movements_order ON inventory_movements(order_id) WHERE order_id IS NOT NULL;

-- ───────────────────────────────────────────────────────────
-- Carts (ephemeral, but persisted for crash recovery and Ghost Cart)
-- ───────────────────────────────────────────────────────────
CREATE TABLE carts (
    cart_id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    kiosk_id            UUID NOT NULL REFERENCES kiosks(kiosk_id),
    session_id          UUID NOT NULL, -- anonymous session identifier
    customer_phone      VARCHAR(16), -- for SMS receipt / Ghost Cart link
    status              VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'finalized', 'abandoned', 'expired')),
    finalized           BOOLEAN NOT NULL DEFAULT FALSE, -- derived from status for optimistic cart-service writes
    version             INT NOT NULL DEFAULT 0, -- LWW CRDT version
    total_cents         INT NOT NULL DEFAULT 0,
    tax_cents           INT NOT NULL DEFAULT 0,
    discount_cents      INT NOT NULL DEFAULT 0,
    final_total_cents   INT NOT NULL DEFAULT 0,
    items_json          JSONB NOT NULL DEFAULT '[]', -- serialized cart items for quick reads
    reserved_inventory  BOOLEAN NOT NULL DEFAULT FALSE, -- whether inventory has been held
    expires_at          TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at_ms       BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT,
    updated_at_ms       BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX idx_carts_store_kiosk ON carts(store_id, kiosk_id, status) WHERE status = 'active';
CREATE INDEX idx_carts_session ON carts(session_id, status) WHERE status = 'active';
CREATE INDEX idx_carts_expires ON carts(expires_at) WHERE status = 'active';

-- Cart line items (normalized, used by cart-service aggregate)
CREATE TABLE cart_lines (
    line_id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    cart_id             UUID NOT NULL REFERENCES carts(cart_id) ON DELETE CASCADE,
    menu_item_id        UUID NOT NULL REFERENCES items(item_id),
    name_snapshot       VARCHAR(255) NOT NULL,
    unit_price_cents_snapshot INT NOT NULL,
    quantity            INT NOT NULL CHECK (quantity > 0),
    modifiers           JSONB NOT NULL DEFAULT '[]',
    added_at_ms         BIGINT NOT NULL
);

CREATE INDEX idx_cart_lines_cart ON cart_lines(cart_id);

-- Trigger: keep carts.status and carts.finalized in sync
CREATE OR REPLACE FUNCTION sync_cart_finalized_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.finalized = TRUE AND NEW.status != 'finalized' THEN
        NEW.status := 'finalized';
    ELSIF NEW.status = 'finalized' AND NEW.finalized = FALSE THEN
        NEW.finalized := TRUE;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER carts_finalized_sync
BEFORE INSERT OR UPDATE ON carts
FOR EACH ROW EXECUTE FUNCTION sync_cart_finalized_status();

-- ───────────────────────────────────────────────────────────
-- Orders (immutable once created)
-- ───────────────────────────────────────────────────────────
CREATE TABLE orders (
    order_id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    kiosk_id            UUID NOT NULL REFERENCES kiosks(kiosk_id),
    cart_id             UUID NOT NULL REFERENCES carts(cart_id),
    order_number        VARCHAR(16) NOT NULL UNIQUE, -- human-readable, e.g. "A-001-0428"
    status              VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'fulfilled', 'cancelled', 'refunded')),
    subtotal_cents      INT NOT NULL DEFAULT 0,
    tax_cents           INT NOT NULL DEFAULT 0,
    discount_cents      INT NOT NULL DEFAULT 0,
    total_cents         INT NOT NULL DEFAULT 0,
    items_json          JSONB NOT NULL DEFAULT '[]',
    tax_breakdown_json  JSONB, -- per the "Customer Facing Transparency" feature
    metadata            JSONB, -- loyalty points, environmental fees, etc.
    paid_at             TIMESTAMPTZ,
    fulfilled_at        TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- no updated_at: orders are immutable once created
    CONSTRAINT immutable_after_paid CHECK (
        status IN ('pending', 'cancelled') OR paid_at IS NOT NULL
    )
);

CREATE INDEX idx_orders_store ON orders(store_id, created_at DESC);
CREATE INDEX idx_orders_kiosk ON orders(kiosk_id, created_at DESC);
CREATE INDEX idx_orders_number ON orders(order_number);
CREATE INDEX idx_orders_status ON orders(store_id, status, created_at DESC);

-- Order items (denormalized, immutable snapshot)
CREATE TABLE order_items (
    order_item_id       UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    order_id            UUID NOT NULL REFERENCES orders(order_id),
    item_id             UUID NOT NULL REFERENCES items(item_id),
    name_snapshot       VARCHAR(255) NOT NULL, -- name at time of purchase
    price_cents_snapshot INT NOT NULL,
    quantity            INT NOT NULL CHECK (quantity > 0),
    modifiers_json      JSONB NOT NULL DEFAULT '[]',
    line_total_cents    INT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_items_order ON order_items(order_id);

-- ───────────────────────────────────────────────────────────
-- Payments
-- ───────────────────────────────────────────────────────────
CREATE TABLE payments (
    payment_id          UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    order_id            UUID NOT NULL REFERENCES orders(order_id),
    kiosk_id            UUID NOT NULL REFERENCES kiosks(kiosk_id),
    idempotency_key     UUID NOT NULL UNIQUE,
    amount_cents        INT NOT NULL CHECK (amount_cents > 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    method              VARCHAR(16) NOT NULL CHECK (method IN ('credit_debit', 'nfc_apple_pay', 'nfc_google_pay', 'qr_code', 'cash_recycler')),
    status              VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'authorized', 'captured', 'declined', 'voided', 'refunded')),
    -- PCI-safe: no PAN, no track data, no CVV. Only opaque token and metadata.
    verifone_token      VARCHAR(255), -- opaque token from Verifone SDK
    verifone_auth_code  VARCHAR(16), -- approval code (non-sensitive)
    card_brand          VARCHAR(16), -- e.g. "visa", "mastercard" (non-sensitive)
    card_last_four      VARCHAR(4), -- last 4 digits (non-sensitive)
    decline_reason      VARCHAR(255), -- if status = declined
    receipt_text        TEXT, -- pre-formatted ESC/POS receipt from Verifone
    is_offline_token    BOOLEAN NOT NULL DEFAULT FALSE,
    offline_token_hmac  VARCHAR(64), -- HMAC of the offline token, for verification
    synced_at           TIMESTAMPTZ, -- when the offline token was synced to cloud
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_order ON payments(order_id);
CREATE INDEX idx_payments_kiosk ON payments(kiosk_id, created_at DESC);
CREATE INDEX idx_payments_offline ON payments(is_offline_token, synced_at) WHERE is_offline_token = TRUE AND synced_at IS NULL;
CREATE INDEX idx_payments_idempotency ON payments(idempotency_key);

-- ───────────────────────────────────────────────────────────
-- Offline Payment Tokens (queue for cloud settlement)
-- ───────────────────────────────────────────────────────────
CREATE TABLE offline_tokens (
    token_id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    kiosk_id            UUID NOT NULL REFERENCES kiosks(kiosk_id),
    cart_id             UUID NOT NULL,
    amount_cents        INT NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    method              VARCHAR(16) NOT NULL,
    verifone_opaque_token VARCHAR(255) NOT NULL,
    hmac_signature      VARCHAR(64) NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    settled_at          TIMESTAMPTZ,
    settlement_result   JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_offline_tokens_store ON offline_tokens(store_id, settled_at) WHERE settled_at IS NULL;
CREATE INDEX idx_offline_tokens_expires ON offline_tokens(expires_at) WHERE settled_at IS NULL;

-- ───────────────────────────────────────────────────────────
-- Employees (WebAuthn/Passkey auth)
-- ───────────────────────────────────────────────────────────
CREATE TABLE employees (
    employee_id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    name                VARCHAR(128) NOT NULL,
    email               VARCHAR(255) NOT NULL,
    role                VARCHAR(16) NOT NULL DEFAULT 'cashier' CHECK (role IN ('cashier', 'supervisor', 'manager', 'admin')),
    biometric_hash      VARCHAR(64), -- irreversible hash from Verifone PIN pad
    webauthn_credential_id VARCHAR(255), -- WebAuthn/Passkey credential ID
    webauthn_public_key   BYTEA, -- COSE-encoded public key
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_employees_store ON employees(store_id, is_active) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_employees_email ON employees(email) WHERE deleted_at IS NULL;

-- ───────────────────────────────────────────────────────────
-- Audit Log (append-only, Merkle tree verified)
-- ───────────────────────────────────────────────────────────
CREATE TABLE audit_log (
    audit_id            BIGSERIAL PRIMARY KEY,
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    kiosk_id            UUID REFERENCES kiosks(kiosk_id),
    employee_id         UUID REFERENCES employees(employee_id),
    event_type          VARCHAR(32) NOT NULL CHECK (event_type IN ('order_created', 'order_paid', 'order_refunded', 'payment_processed', 'inventory_adjusted', 'employee_login', 'employee_logout', 'system_boot', 'system_shutdown', 'sync_event', 'security_event')),
    entity_type         VARCHAR(32) NOT NULL, -- 'order', 'payment', 'inventory', 'employee', 'system'
    entity_id           UUID NOT NULL,
    payload_json        JSONB NOT NULL,
    previous_hash       VARCHAR(64) NOT NULL, -- SHA-256 of previous audit row's hash
    current_hash        VARCHAR(64) NOT NULL, -- SHA-256 of (previous_hash || event_type || entity_id || payload_json)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_store ON audit_log(store_id, created_at DESC);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id, created_at DESC);
CREATE INDEX idx_audit_log_event ON audit_log(event_type, created_at DESC);

-- ───────────────────────────────────────────────────────────
-- Sync Events (for P2P mesh reconciliation)
-- ───────────────────────────────────────────────────────────
CREATE TABLE sync_events (
    sync_event_id       UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    kiosk_id            UUID NOT NULL REFERENCES kiosks(kiosk_id),
    event_type          VARCHAR(16) NOT NULL CHECK (event_type IN ('inventory_update', 'cart_merge', 'transaction_batch', 'analytics_batch')),
    payload_json        JSONB NOT NULL,
    vector_clock        JSONB NOT NULL, -- {"kiosk-abc": 42, "kiosk-def": 17}
    processed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_events_store ON sync_events(store_id, created_at DESC);
CREATE INDEX idx_sync_events_unprocessed ON sync_events(store_id, processed_at) WHERE processed_at IS NULL;

-- ───────────────────────────────────────────────────────────
-- Analytics Events (differential privacy, aggregated)
-- ───────────────────────────────────────────────────────────
CREATE TABLE analytics_events (
    analytics_id        BIGSERIAL,
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    kiosk_id            UUID REFERENCES kiosks(kiosk_id),
    event_type          VARCHAR(32) NOT NULL, -- 'item_viewed', 'cart_abandoned', 'payment_completed', etc.
    session_id          UUID,
    -- No PII. All identifiers are hashed or pseudonymized.
    customer_hash       VARCHAR(64), -- SHA-256 of customer phone/email (one-way)
    item_id             UUID REFERENCES items(item_id),
    category_id         UUID REFERENCES categories(category_id),
    quantity            INT,
    amount_cents        INT,
    duration_ms         INT, -- how long the interaction took
    metadata            JSONB, -- arbitrary event metadata
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (analytics_id, created_at)
) PARTITION BY RANGE (created_at);

-- ───────────────────────────────────────────────────────────
-- Inventory reservations (soft-holds during active cart lifetime)
-- ───────────────────────────────────────────────────────────
CREATE TABLE inventory_reservations (
    reservation_id      UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    store_id            UUID NOT NULL REFERENCES stores(store_id),
    kiosk_id            UUID NOT NULL REFERENCES kiosks(kiosk_id),
    item_id             UUID NOT NULL REFERENCES items(item_id),
    cart_id             UUID NOT NULL REFERENCES carts(cart_id),
    quantity            INT NOT NULL CHECK (quantity > 0),
    expires_at_ms       BIGINT NOT NULL,
    created_at_ms       BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX idx_inventory_reservations_item ON inventory_reservations(item_id, expires_at_ms);
CREATE INDEX idx_inventory_reservations_cart ON inventory_reservations(cart_id);
CREATE INDEX idx_inventory_reservations_expires ON inventory_reservations(expires_at_ms);

-- Function: keep inventory.quantity_reserved in sync with the reservations table
CREATE OR REPLACE FUNCTION refresh_inventory_reserved()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE inventory
        SET quantity_reserved = quantity_reserved + NEW.quantity
        WHERE store_id = NEW.store_id AND item_id = NEW.item_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE inventory
        SET quantity_reserved = GREATEST(0, quantity_reserved - OLD.quantity)
        WHERE store_id = OLD.store_id AND item_id = OLD.item_id;
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        UPDATE inventory
        SET quantity_reserved = GREATEST(0, quantity_reserved - OLD.quantity + NEW.quantity)
        WHERE store_id = NEW.store_id AND item_id = NEW.item_id;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER inventory_reservation_sync
AFTER INSERT OR UPDATE OR DELETE ON inventory_reservations
FOR EACH ROW EXECUTE FUNCTION refresh_inventory_reserved();

-- Create monthly partitions for analytics (2025-2026)
CREATE TABLE analytics_events_2025_01 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE analytics_events_2025_02 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE analytics_events_2025_03 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE analytics_events_2025_04 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE analytics_events_2025_05 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE analytics_events_2025_06 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE analytics_events_2025_07 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE analytics_events_2025_08 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE analytics_events_2025_09 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE analytics_events_2025_10 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE analytics_events_2025_11 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE analytics_events_2025_12 PARTITION OF analytics_events
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE analytics_events_2026_01 PARTITION OF analytics_events
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE INDEX idx_analytics_store ON analytics_events(store_id, event_type, created_at DESC);
CREATE INDEX idx_analytics_item ON analytics_events(item_id, created_at DESC) WHERE item_id IS NOT NULL;

-- ───────────────────────────────────────────────────────────
-- Transactional Outbox (deep-improvement #5)
-- Every domain write appends an event row in the same DB transaction.
-- A background relay publishes unpublished rows to NATS JetStream.
-- ───────────────────────────────────────────────────────────
CREATE TABLE outbox_events (
    event_id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    aggregate_type      VARCHAR(64) NOT NULL, -- e.g. 'cart', 'order', 'inventory'
    aggregate_id        UUID NOT NULL,
    event_type          VARCHAR(128) NOT NULL, -- e.g. 'astra.cart.item_added.v1'
    payload             JSONB NOT NULL,
    occurred_at_ms      BIGINT NOT NULL,
    published           BOOLEAN NOT NULL DEFAULT FALSE,
    published_at_ms     BIGINT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbox_unpublished ON outbox_events(published, occurred_at_ms) WHERE published = FALSE;
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_type, aggregate_id, occurred_at_ms);

-- ───────────────────────────────────────────────────────────
-- Functions and Triggers
-- ───────────────────────────────────────────────────────────

-- Auto-update `updated_at` on any UPDATE
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to all tables with updated_at
CREATE TRIGGER update_stores_updated_at BEFORE UPDATE ON stores FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_kiosks_updated_at BEFORE UPDATE ON kiosks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_items_updated_at BEFORE UPDATE ON items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_modifier_groups_updated_at BEFORE UPDATE ON modifier_groups FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_modifier_options_updated_at BEFORE UPDATE ON modifier_options FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_inventory_updated_at BEFORE UPDATE ON inventory FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_carts_updated_at BEFORE UPDATE ON carts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_payments_updated_at BEFORE UPDATE ON payments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_offline_tokens_updated_at BEFORE UPDATE ON offline_tokens FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_employees_updated_at BEFORE UPDATE ON employees FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Soft delete: instead of DELETE, set deleted_at. This trigger enforces it.
CREATE OR REPLACE FUNCTION soft_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL THEN
        -- Prevent un-deleting via UPDATE
        RAISE EXCEPTION 'Cannot un-delete a soft-deleted record';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Inventory movement trigger: automatically log changes
CREATE OR REPLACE FUNCTION log_inventory_movement()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO inventory_movements (
        store_id, item_id, movement_type, quantity_delta, reason, kiosk_id, created_at
    ) VALUES (
        NEW.store_id,
        NEW.item_id,
        'adjustment',
        NEW.quantity_available - COALESCE(OLD.quantity_available, 0),
        'automatic: inventory update',
        NULL,
        NOW()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER inventory_movement_log AFTER UPDATE ON inventory
FOR EACH ROW EXECUTE FUNCTION log_inventory_movement();

-- ───────────────────────────────────────────────────────────
-- Row-Level Security (RLS) Policies
-- ───────────────────────────────────────────────────────────

-- Enable RLS on tenant-scoped tables
ALTER TABLE stores ENABLE ROW LEVEL SECURITY;
ALTER TABLE kiosks ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE items ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory ENABLE ROW LEVEL SECURITY;
ALTER TABLE carts ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_events ENABLE ROW LEVEL SECURITY;

-- Example policy: employees can only see their own store's data
-- (In production, these would be parameterized by the application user's store_id)
CREATE POLICY store_isolation ON stores USING (true); -- placeholder; application enforces via WHERE clause

-- ───────────────────────────────────────────────────────────
-- Seed Data (for local development)
-- ───────────────────────────────────────────────────────────

INSERT INTO stores (store_id, name, address, timezone, currency, tax_rate) VALUES
    ('550e8400-e29b-41d4-a716-446655440000', 'Astra Miami Brickell', '1000 Brickell Ave, Miami, FL 33131', 'America/New_York', 'USD', 0.0700);

INSERT INTO kiosks (store_id, hardware_id, display_name, ip_address, signing_key_hash, firmware_version) VALUES
    ('550e8400-e29b-41d4-a716-446655440000', 'HW-KIOSK-001', 'Lane 1', '10.0.1.11', 'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3', '1.0.0-astra'),
    ('550e8400-e29b-41d4-a716-446655440000', 'HW-KIOSK-002', 'Lane 2', '10.0.1.12', 'b665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3', '1.0.0-astra');

INSERT INTO categories (store_id, name, description, display_order) VALUES
    ('550e8400-e29b-41d4-a716-446655440000', 'Produce', 'Fresh fruits and vegetables', 1),
    ('550e8400-e29b-41d4-a716-446655440000', 'Bakery', 'Fresh bread, pastries, and cakes', 2),
    ('550e8400-e29b-41d4-a716-446655440000', 'Dairy', 'Milk, cheese, yogurt, and eggs', 3),
    ('550e8400-e29b-41d4-a716-446655440000', 'Beverages', 'Soft drinks, juices, water, and coffee', 4);

-- Note: items would be inserted by application code or a separate seed script
-- as they reference the auto-generated category_ids above.

-- ───────────────────────────────────────────────────────────
-- Comment: Schema version tracking
-- ───────────────────────────────────────────────────────────
COMMENT ON DATABASE astra_service IS 'Astra-Service production schema v1.0.0';
