-- 0003_seed_permissions.sql
-- Seed global permissions and a system tenant with standard RBAC roles.

-- ---------------------------------------------------------------------------
-- UP
-- ---------------------------------------------------------------------------

-- System tenant that owns global roles.
INSERT INTO tenants (slug, name, billing_email, plan, is_active)
VALUES ('system', 'System Tenant', 'system@astra.local', 'enterprise', TRUE)
ON CONFLICT (slug) DO NOTHING;

-- Global permissions.
INSERT INTO permissions (resource, action, description) VALUES
  ('tenant', 'read', 'View tenant details'),
  ('tenant', 'write', 'Create and update tenants'),
  ('tenant', 'delete', 'Delete tenants'),
  ('location', 'read', 'View locations'),
  ('location', 'write', 'Create and update locations'),
  ('location', 'delete', 'Delete locations'),
  ('lane', 'read', 'View lanes'),
  ('lane', 'write', 'Create and update lanes'),
  ('lane', 'delete', 'Delete lanes'),
  ('store', 'read', 'View stores'),
  ('store', 'write', 'Create and update stores'),
  ('store', 'delete', 'Delete stores'),
  ('kiosk', 'read', 'View kiosks'),
  ('kiosk', 'write', 'Create and update kiosks'),
  ('kiosk', 'delete', 'Delete kiosks'),
  ('kiosk', 'control', 'Control kiosk state'),
  ('category', 'read', 'View categories'),
  ('category', 'write', 'Create and update categories'),
  ('category', 'delete', 'Delete categories'),
  ('item', 'read', 'View items'),
  ('item', 'write', 'Create and update items'),
  ('item', 'delete', 'Delete items'),
  ('modifier_group', 'read', 'View modifier groups'),
  ('modifier_group', 'write', 'Create and update modifier groups'),
  ('modifier_group', 'delete', 'Delete modifier groups'),
  ('inventory', 'read', 'View inventory'),
  ('inventory', 'write', 'Update inventory'),
  ('inventory', 'adjust', 'Adjust inventory balances'),
  ('cart', 'read', 'View carts'),
  ('cart', 'write', 'Create and update carts'),
  ('cart', 'delete', 'Delete carts'),
  ('order', 'read', 'View orders'),
  ('order', 'write', 'Create and update orders'),
  ('order', 'refund', 'Refund orders'),
  ('order', 'cancel', 'Cancel orders'),
  ('payment', 'read', 'View payments'),
  ('payment', 'process', 'Process payments'),
  ('payment', 'refund', 'Process refunds'),
  ('refund', 'read', 'View refunds'),
  ('refund', 'process', 'Process refund requests'),
  ('offline_token', 'read', 'View offline tokens'),
  ('offline_token', 'settle', 'Settle offline tokens'),
  ('employee', 'read', 'View employees'),
  ('employee', 'write', 'Create and update employees'),
  ('employee', 'delete', 'Delete employees'),
  ('user', 'read', 'View users'),
  ('user', 'write', 'Create and update users'),
  ('user', 'delete', 'Delete users'),
  ('role', 'read', 'View roles'),
  ('role', 'write', 'Create and update roles'),
  ('role', 'delete', 'Delete roles'),
  ('permission', 'read', 'View permissions'),
  ('audit_log', 'read', 'View audit logs'),
  ('analytics_event', 'read', 'View analytics events'),
  ('event_store', 'read', 'View event store'),
  ('sync_event', 'read', 'View sync events'),
  ('sync_event', 'process', 'Process sync events'),
  ('system', 'configure', 'Configure system settings')
ON CONFLICT (resource, action) DO NOTHING;

-- System roles owned by the system tenant.
WITH system_tenant AS (
  SELECT tenant_id FROM tenants WHERE slug = 'system'
)
INSERT INTO roles (tenant_id, name, description, is_system)
SELECT tenant_id, 'Admin', 'Full system access', TRUE FROM system_tenant
UNION ALL
SELECT tenant_id, 'Manager', 'Store and location management', TRUE FROM system_tenant
UNION ALL
SELECT tenant_id, 'Operator', 'Day-to-day operations', TRUE FROM system_tenant
UNION ALL
SELECT tenant_id, 'Viewer', 'Read-only access', TRUE FROM system_tenant
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Admin gets every permission.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'Admin' AND r.is_system = TRUE
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Manager permissions: broad access except destructive/system-level actions.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'Manager' AND r.is_system = TRUE
  AND NOT (
    (p.resource = 'tenant' AND p.action = 'delete')
    OR (p.resource = 'permission')
    OR (p.resource = 'system' AND p.action = 'configure')
    OR (p.resource = 'user' AND p.action = 'delete')
    OR (p.resource = 'role' AND p.action = 'delete')
    OR (p.resource = 'employee' AND p.action = 'delete')
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Operator permissions: operational actions only.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'Operator' AND r.is_system = TRUE
  AND (
    p.action = 'read'
    OR (p.resource = 'cart' AND p.action IN ('write', 'delete'))
    OR (p.resource = 'order' AND p.action IN ('write', 'refund', 'cancel'))
    OR (p.resource = 'payment' AND p.action IN ('process', 'refund'))
    OR (p.resource = 'refund' AND p.action = 'process')
    OR (p.resource = 'offline_token' AND p.action = 'settle')
    OR (p.resource = 'sync_event' AND p.action = 'process')
    OR (p.resource = 'inventory' AND p.action IN ('write', 'adjust'))
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Viewer permissions: read-only across all resources.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'Viewer' AND r.is_system = TRUE
  AND p.action = 'read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- DOWN
-- ---------------------------------------------------------------------------

DELETE FROM role_permissions
WHERE role_id IN (SELECT role_id FROM roles WHERE is_system = TRUE);

DELETE FROM roles WHERE is_system = TRUE;

DELETE FROM permissions;

DELETE FROM tenants WHERE slug = 'system';
