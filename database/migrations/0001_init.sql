-- 0001_init.sql
-- Initial schema for the Astra-Service platform.
-- Generated from database/schemas/drizzle.ts

-- ---------------------------------------------------------------------------
-- UP
-- ---------------------------------------------------------------------------

-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Enums
CREATE TYPE tenant_plan AS ENUM ('standard', 'enterprise');
CREATE TYPE kiosk_sync_status AS ENUM ('online', 'offline', 'degraded', 'maintenance');
CREATE TYPE item_tax_category AS ENUM ('standard', 'exempt', 'reduced');
CREATE TYPE weight_unit AS ENUM ('g', 'kg', 'lb', 'oz');
CREATE TYPE cart_status AS ENUM ('active', 'finalized', 'abandoned', 'expired');
CREATE TYPE order_status AS ENUM ('pending', 'paid', 'fulfilled', 'cancelled', 'refunded');
CREATE TYPE payment_method AS ENUM ('credit_debit', 'nfc_apple_pay', 'nfc_google_pay', 'qr_code', 'cash_recycler');
CREATE TYPE payment_status AS ENUM ('pending', 'authorized', 'captured', 'declined', 'voided', 'refunded');
CREATE TYPE employee_role AS ENUM ('cashier', 'supervisor', 'manager', 'admin');
CREATE TYPE audit_event_type AS ENUM (
  'order_created',
  'order_paid',
  'order_refunded',
  'payment_processed',
  'inventory_adjusted',
  'employee_login',
  'employee_logout',
  'system_boot',
  'system_shutdown',
  'sync_event',
  'security_event'
);
CREATE TYPE inventory_transaction_type AS ENUM ('sale', 'restock', 'adjustment', 'reserved', 'released', 'waste', 'return');
CREATE TYPE sync_event_type AS ENUM ('inventory_update', 'cart_merge', 'transaction_batch', 'analytics_batch');
CREATE TYPE refund_status AS ENUM ('pending', 'completed', 'failed');

-- Trigger function for updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- Tenant / Location / Lane hierarchy
-- ---------------------------------------------------------------------------

CREATE TABLE tenants (
  tenant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  billing_email VARCHAR(255) NOT NULL,
  plan tenant_plan NOT NULL DEFAULT 'standard',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT tenants_slug_unique UNIQUE (slug)
);
CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_tenants_set_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE locations (
  location_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
  slug VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  address TEXT,
  timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
  currency VARCHAR(3) NOT NULL DEFAULT 'USD',
  tax_rate DECIMAL(5,4) NOT NULL DEFAULT 0.0000,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT locations_tenant_slug_unique UNIQUE (tenant_id, slug)
);
CREATE INDEX idx_locations_tenant ON locations(tenant_id, deleted_at) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_locations_set_updated_at
BEFORE UPDATE ON locations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE lanes (
  lane_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(location_id),
  display_name VARCHAR(64) NOT NULL,
  lane_number INTEGER NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT lanes_location_number_unique UNIQUE (location_id, lane_number)
);
CREATE INDEX idx_lanes_location ON lanes(location_id, deleted_at) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_lanes_set_updated_at
BEFORE UPDATE ON lanes
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Stores / Kiosks
-- ---------------------------------------------------------------------------

CREATE TABLE stores (
  store_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID REFERENCES tenants(tenant_id),
  location_id UUID REFERENCES locations(location_id),
  name VARCHAR(255) NOT NULL,
  address TEXT,
  timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
  currency VARCHAR(3) NOT NULL DEFAULT 'USD',
  tax_rate DECIMAL(5,4) NOT NULL DEFAULT 0.0000,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_stores_deleted_at ON stores(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_stores_tenant ON stores(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_stores_location ON stores(location_id, deleted_at) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_stores_set_updated_at
BEFORE UPDATE ON stores
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE employees (
  employee_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  name VARCHAR(128) NOT NULL,
  email VARCHAR(255) NOT NULL,
  role employee_role NOT NULL DEFAULT 'cashier',
  biometric_hash VARCHAR(64),
  webauthn_credential_id VARCHAR(255),
  webauthn_public_key BYTEA,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_employees_store ON employees(store_id, is_active) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_employees_email ON employees(email) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_employees_set_updated_at
BEFORE UPDATE ON employees
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE kiosks (
  kiosk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  lane_id UUID REFERENCES lanes(lane_id),
  tenant_id UUID REFERENCES tenants(tenant_id),
  hardware_id VARCHAR(64) NOT NULL,
  display_name VARCHAR(64) NOT NULL,
  ip_address INET,
  last_seen_at TIMESTAMPTZ,
  sync_status kiosk_sync_status NOT NULL DEFAULT 'online',
  is_leader BOOLEAN NOT NULL DEFAULT FALSE,
  signing_key_hash VARCHAR(64) NOT NULL,
  firmware_version VARCHAR(32),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT kiosks_hardware_id_unique UNIQUE (hardware_id)
);
CREATE INDEX idx_kiosks_store ON kiosks(store_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_kiosks_leader ON kiosks(store_id, is_leader) WHERE is_leader = TRUE AND deleted_at IS NULL;
CREATE UNIQUE INDEX idx_kiosks_one_leader_per_store ON kiosks(store_id) WHERE is_leader = TRUE AND deleted_at IS NULL;
CREATE INDEX idx_kiosks_lane ON kiosks(lane_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_kiosks_tenant ON kiosks(tenant_id, deleted_at) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_kiosks_set_updated_at
BEFORE UPDATE ON kiosks
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Menu: Categories / Items / Modifiers
-- ---------------------------------------------------------------------------

CREATE TABLE categories (
  category_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  parent_id UUID REFERENCES categories(category_id),
  name VARCHAR(128) NOT NULL,
  description TEXT,
  display_order INTEGER NOT NULL DEFAULT 0,
  image_url VARCHAR(512),
  blurhash VARCHAR(32),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_categories_store ON categories(store_id, display_order, is_active) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_categories_set_updated_at
BEFORE UPDATE ON categories
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE items (
  item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  category_id UUID NOT NULL REFERENCES categories(category_id),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  price_cents INTEGER NOT NULL,
  cost_cents INTEGER,
  plu VARCHAR(16),
  barcode VARCHAR(32),
  sku VARCHAR(64),
  image_url VARCHAR(512),
  blurhash VARCHAR(32),
  tax_category item_tax_category NOT NULL DEFAULT 'standard',
  is_weight_based BOOLEAN NOT NULL DEFAULT FALSE,
  weight_unit weight_unit,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_items_store_category ON items(store_id, category_id, is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_items_barcode ON items(barcode) WHERE barcode IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_items_plu ON items(plu) WHERE plu IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_items_name_trgm ON items USING gin (name gin_trgm_ops);

CREATE TRIGGER trg_items_set_updated_at
BEFORE UPDATE ON items
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE modifier_groups (
  modifier_group_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  name VARCHAR(128) NOT NULL,
  description TEXT,
  min_select INTEGER NOT NULL DEFAULT 0,
  max_select INTEGER NOT NULL DEFAULT 1,
  display_order INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE TRIGGER trg_modifier_groups_set_updated_at
BEFORE UPDATE ON modifier_groups
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE modifier_options (
  modifier_option_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  modifier_group_id UUID NOT NULL REFERENCES modifier_groups(modifier_group_id),
  name VARCHAR(128) NOT NULL,
  price_delta_cents INTEGER NOT NULL DEFAULT 0,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  display_order INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE TRIGGER trg_modifier_options_set_updated_at
BEFORE UPDATE ON modifier_options
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE item_modifier_groups (
  item_id UUID NOT NULL REFERENCES items(item_id),
  modifier_group_id UUID NOT NULL REFERENCES modifier_groups(modifier_group_id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (item_id, modifier_group_id)
);

-- ---------------------------------------------------------------------------
-- Carts
-- ---------------------------------------------------------------------------

CREATE TABLE carts (
  cart_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  kiosk_id UUID NOT NULL REFERENCES kiosks(kiosk_id),
  session_id UUID NOT NULL,
  customer_phone VARCHAR(16),
  status cart_status NOT NULL DEFAULT 'active',
  finalized BOOLEAN NOT NULL DEFAULT FALSE,
  version INTEGER NOT NULL DEFAULT 0,
  total_cents INTEGER NOT NULL DEFAULT 0,
  tax_cents INTEGER NOT NULL DEFAULT 0,
  discount_cents INTEGER NOT NULL DEFAULT 0,
  final_total_cents INTEGER NOT NULL DEFAULT 0,
  items_json JSONB NOT NULL DEFAULT '[]',
  reserved_inventory BOOLEAN NOT NULL DEFAULT FALSE,
  expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '10 minutes'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at_ms BIGINT NOT NULL DEFAULT ((EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT),
  updated_at_ms BIGINT NOT NULL DEFAULT ((EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT)
);
CREATE INDEX idx_carts_store_kiosk ON carts(store_id, kiosk_id, status) WHERE status = 'active';
CREATE INDEX idx_carts_session ON carts(session_id, status) WHERE status = 'active';
CREATE INDEX idx_carts_expires ON carts(expires_at) WHERE status = 'active';

CREATE TRIGGER trg_carts_set_updated_at
BEFORE UPDATE ON carts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cart_lines (
  line_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cart_id UUID NOT NULL REFERENCES carts(cart_id) ON DELETE CASCADE,
  menu_item_id UUID NOT NULL REFERENCES items(item_id),
  name_snapshot VARCHAR(255) NOT NULL,
  unit_price_cents_snapshot INTEGER NOT NULL,
  quantity INTEGER NOT NULL,
  modifiers JSONB NOT NULL DEFAULT '[]',
  added_at_ms BIGINT NOT NULL
);
CREATE INDEX idx_cart_lines_cart ON cart_lines(cart_id);

-- ---------------------------------------------------------------------------
-- Inventory
-- ---------------------------------------------------------------------------

CREATE TABLE inventory (
  inventory_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  item_id UUID NOT NULL REFERENCES items(item_id),
  quantity_available INTEGER NOT NULL DEFAULT 0,
  quantity_reserved INTEGER NOT NULL DEFAULT 0,
  quantity_on_order INTEGER NOT NULL DEFAULT 0,
  reorder_point INTEGER NOT NULL DEFAULT 0,
  reorder_quantity INTEGER NOT NULL DEFAULT 0,
  location VARCHAR(64),
  last_counted_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT inventory_store_item_unique UNIQUE (store_id, item_id)
);
CREATE INDEX idx_inventory_store ON inventory(store_id);
CREATE INDEX idx_inventory_low_stock ON inventory(store_id, quantity_available) WHERE quantity_available <= reorder_point;

CREATE TRIGGER trg_inventory_set_updated_at
BEFORE UPDATE ON inventory
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE inventory_transactions (
  transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  item_id UUID NOT NULL REFERENCES items(item_id),
  transaction_type inventory_transaction_type NOT NULL,
  quantity_delta INTEGER NOT NULL,
  running_balance INTEGER NOT NULL,
  reference_id UUID,
  reference_type VARCHAR(32),
  kiosk_id UUID REFERENCES kiosks(kiosk_id),
  employee_id UUID REFERENCES employees(employee_id),
  notes VARCHAR(500),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_inventory_transactions_store_item ON inventory_transactions(store_id, item_id, created_at);
CREATE INDEX idx_inventory_transactions_reference ON inventory_transactions(reference_type, reference_id);

CREATE TABLE inventory_reservations (
  reservation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  kiosk_id UUID NOT NULL REFERENCES kiosks(kiosk_id),
  item_id UUID NOT NULL REFERENCES items(item_id),
  cart_id UUID NOT NULL REFERENCES carts(cart_id),
  quantity INTEGER NOT NULL,
  expires_at_ms BIGINT NOT NULL,
  created_at_ms BIGINT NOT NULL
);
CREATE INDEX idx_inventory_reservations_item ON inventory_reservations(item_id, expires_at_ms);
CREATE INDEX idx_inventory_reservations_cart ON inventory_reservations(cart_id);
CREATE INDEX idx_inventory_reservations_expires ON inventory_reservations(expires_at_ms);

-- ---------------------------------------------------------------------------
-- Orders
-- ---------------------------------------------------------------------------

CREATE TABLE orders (
  order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  kiosk_id UUID NOT NULL REFERENCES kiosks(kiosk_id),
  cart_id UUID NOT NULL REFERENCES carts(cart_id),
  order_number VARCHAR(16) NOT NULL,
  status order_status NOT NULL DEFAULT 'pending',
  subtotal_cents INTEGER NOT NULL DEFAULT 0,
  tax_cents INTEGER NOT NULL DEFAULT 0,
  discount_cents INTEGER NOT NULL DEFAULT 0,
  total_cents INTEGER NOT NULL DEFAULT 0,
  items_json JSONB NOT NULL DEFAULT '[]',
  tax_breakdown_json JSONB,
  metadata JSONB,
  paid_at TIMESTAMPTZ,
  fulfilled_at TIMESTAMPTZ,
  cancelled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT orders_order_number_unique UNIQUE (order_number)
);
CREATE INDEX idx_orders_store ON orders(store_id, created_at);
CREATE INDEX idx_orders_kiosk ON orders(kiosk_id, created_at);
CREATE INDEX idx_orders_number ON orders(order_number);
CREATE INDEX idx_orders_status ON orders(store_id, status, created_at);

CREATE TABLE order_items (
  order_item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL REFERENCES orders(order_id),
  item_id UUID NOT NULL REFERENCES items(item_id),
  name_snapshot VARCHAR(255) NOT NULL,
  price_cents_snapshot INTEGER NOT NULL,
  quantity INTEGER NOT NULL,
  modifiers_json JSONB NOT NULL DEFAULT '[]',
  line_total_cents INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_order_items_order ON order_items(order_id);

-- ---------------------------------------------------------------------------
-- Payments / Refunds / Offline Tokens
-- ---------------------------------------------------------------------------

CREATE TABLE payments (
  payment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL REFERENCES orders(order_id),
  kiosk_id UUID NOT NULL REFERENCES kiosks(kiosk_id),
  idempotency_key UUID NOT NULL,
  amount_cents INTEGER NOT NULL,
  currency VARCHAR(3) NOT NULL DEFAULT 'USD',
  method payment_method NOT NULL,
  status payment_status NOT NULL DEFAULT 'pending',
  verifone_token VARCHAR(255),
  verifone_auth_code VARCHAR(16),
  card_brand VARCHAR(16),
  card_last_four VARCHAR(4),
  decline_reason VARCHAR(255),
  receipt_text TEXT,
  is_offline_token BOOLEAN NOT NULL DEFAULT FALSE,
  offline_token_hmac VARCHAR(64),
  synced_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT payments_idempotency_key_unique UNIQUE (idempotency_key)
);
CREATE INDEX idx_payments_order ON payments(order_id);
CREATE INDEX idx_payments_kiosk ON payments(kiosk_id, created_at);
CREATE INDEX idx_payments_offline ON payments(is_offline_token, synced_at) WHERE is_offline_token = TRUE AND synced_at IS NULL;
CREATE INDEX idx_payments_idempotency ON payments(idempotency_key);

CREATE TRIGGER trg_payments_set_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE refunds (
  refund_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  payment_id UUID NOT NULL REFERENCES payments(payment_id),
  order_id UUID NOT NULL REFERENCES orders(order_id),
  kiosk_id UUID NOT NULL REFERENCES kiosks(kiosk_id),
  amount_cents INTEGER NOT NULL,
  currency VARCHAR(3) NOT NULL DEFAULT 'USD',
  reason VARCHAR(255) NOT NULL,
  status refund_status NOT NULL DEFAULT 'pending',
  verifone_reference VARCHAR(255),
  processed_by UUID REFERENCES employees(employee_id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_refunds_payment ON refunds(payment_id);
CREATE INDEX idx_refunds_order ON refunds(order_id, created_at);

CREATE TRIGGER trg_refunds_set_updated_at
BEFORE UPDATE ON refunds
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE offline_tokens (
  token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  kiosk_id UUID NOT NULL REFERENCES kiosks(kiosk_id),
  cart_id UUID NOT NULL,
  amount_cents INTEGER NOT NULL,
  currency VARCHAR(3) NOT NULL DEFAULT 'USD',
  method VARCHAR(16) NOT NULL,
  verifone_opaque_token VARCHAR(255) NOT NULL,
  hmac_signature VARCHAR(64) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  settled_at TIMESTAMPTZ,
  settlement_result JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_offline_tokens_store ON offline_tokens(store_id, settled_at) WHERE settled_at IS NULL;
CREATE INDEX idx_offline_tokens_expires ON offline_tokens(expires_at) WHERE settled_at IS NULL;

CREATE TRIGGER trg_offline_tokens_set_updated_at
BEFORE UPDATE ON offline_tokens
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Users / Roles / Permissions
-- ---------------------------------------------------------------------------

CREATE TABLE roles (
  role_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
  name VARCHAR(64) NOT NULL,
  description TEXT,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT roles_tenant_name_unique UNIQUE (tenant_id, name)
);
CREATE INDEX idx_roles_tenant ON roles(tenant_id);

CREATE TRIGGER trg_roles_set_updated_at
BEFORE UPDATE ON roles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE permissions (
  permission_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  resource VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  description TEXT,
  CONSTRAINT permissions_resource_action_unique UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
  role_id UUID NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(permission_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
  user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
  email VARCHAR(255) NOT NULL,
  name VARCHAR(128) NOT NULL,
  role_id UUID NOT NULL REFERENCES roles(role_id),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  webauthn_credential_id VARCHAR(255),
  webauthn_public_key BYTEA,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT users_tenant_email_unique UNIQUE (tenant_id, email)
);
CREATE INDEX idx_users_tenant ON users(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role_id);

CREATE TRIGGER trg_users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Audit / Event Store / Outbox / Sync / Analytics
-- ---------------------------------------------------------------------------

CREATE TABLE audit_logs (
  audit_id BIGINT NOT NULL,
  store_id UUID NOT NULL REFERENCES stores(store_id),
  tenant_id UUID REFERENCES tenants(tenant_id),
  lane_id UUID REFERENCES lanes(lane_id),
  kiosk_id UUID REFERENCES kiosks(kiosk_id),
  employee_id UUID REFERENCES employees(employee_id),
  user_id UUID REFERENCES users(user_id),
  event_type audit_event_type NOT NULL,
  entity_type VARCHAR(32) NOT NULL,
  entity_id UUID NOT NULL,
  payload_json JSONB NOT NULL,
  previous_hash VARCHAR(64) NOT NULL,
  current_hash VARCHAR(64) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (audit_id, created_at)
) PARTITION BY RANGE (created_at);
CREATE INDEX idx_audit_logs_store ON audit_logs(store_id, created_at);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at);
CREATE INDEX idx_audit_logs_event ON audit_logs(event_type, created_at);

CREATE TABLE event_store (
  event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_schema VARCHAR(64) NOT NULL,
  aggregate_type VARCHAR(64) NOT NULL,
  aggregate_id UUID NOT NULL,
  sequence_number BIGINT NOT NULL,
  payload JSONB NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}',
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT event_store_aggregate_unique UNIQUE (aggregate_type, aggregate_id, sequence_number)
);
CREATE INDEX idx_event_store_aggregate ON event_store(aggregate_type, aggregate_id, sequence_number);
CREATE INDEX idx_event_store_occurred ON event_store(occurred_at);

CREATE TABLE outbox_events (
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
CREATE INDEX idx_outbox_unpublished ON outbox_events(published, occurred_at_ms) WHERE published = FALSE;
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_type, aggregate_id, occurred_at_ms);

CREATE TABLE sync_events (
  sync_event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id UUID NOT NULL REFERENCES stores(store_id),
  kiosk_id UUID NOT NULL REFERENCES kiosks(kiosk_id),
  event_type sync_event_type NOT NULL,
  payload_json JSONB NOT NULL,
  vector_clock JSONB NOT NULL,
  processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sync_events_store ON sync_events(store_id, created_at);
CREATE INDEX idx_sync_events_unprocessed ON sync_events(store_id, processed_at) WHERE processed_at IS NULL;

CREATE TABLE analytics_events (
  analytics_id BIGINT NOT NULL,
  store_id UUID NOT NULL REFERENCES stores(store_id),
  kiosk_id UUID REFERENCES kiosks(kiosk_id),
  event_type VARCHAR(32) NOT NULL,
  session_id UUID,
  customer_hash VARCHAR(64),
  item_id UUID REFERENCES items(item_id),
  category_id UUID REFERENCES categories(category_id),
  quantity INTEGER,
  amount_cents INTEGER,
  duration_ms INTEGER,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (analytics_id, created_at)
) PARTITION BY RANGE (created_at);
CREATE INDEX idx_analytics_store ON analytics_events(store_id, event_type, created_at);
CREATE INDEX idx_analytics_item ON analytics_events(item_id, created_at) WHERE item_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- DOWN
-- ---------------------------------------------------------------------------

DROP TABLE IF EXISTS analytics_events CASCADE;
DROP TABLE IF EXISTS sync_events CASCADE;
DROP TABLE IF EXISTS outbox_events CASCADE;
DROP TABLE IF EXISTS event_store CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS offline_tokens CASCADE;
DROP TABLE IF EXISTS refunds CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS inventory_reservations CASCADE;
DROP TABLE IF EXISTS inventory_transactions CASCADE;
DROP TABLE IF EXISTS inventory CASCADE;
DROP TABLE IF EXISTS cart_lines CASCADE;
DROP TABLE IF EXISTS carts CASCADE;
DROP TABLE IF EXISTS item_modifier_groups CASCADE;
DROP TABLE IF EXISTS modifier_options CASCADE;
DROP TABLE IF EXISTS modifier_groups CASCADE;
DROP TABLE IF EXISTS items CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS kiosks CASCADE;
DROP TABLE IF EXISTS employees CASCADE;
DROP TABLE IF EXISTS stores CASCADE;
DROP TABLE IF EXISTS lanes CASCADE;
DROP TABLE IF EXISTS locations CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

DROP FUNCTION IF EXISTS set_updated_at() CASCADE;

DROP TYPE IF EXISTS refund_status;
DROP TYPE IF EXISTS sync_event_type;
DROP TYPE IF EXISTS inventory_transaction_type;
DROP TYPE IF EXISTS audit_event_type;
DROP TYPE IF EXISTS employee_role;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_method;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS cart_status;
DROP TYPE IF EXISTS weight_unit;
DROP TYPE IF EXISTS item_tax_category;
DROP TYPE IF EXISTS kiosk_sync_status;
DROP TYPE IF EXISTS tenant_plan;
