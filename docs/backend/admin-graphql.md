# Admin GraphQL

## Overview

The Admin GraphQL service (`services/admin-graphql/`) provides a GraphQL API for the kiosk admin dashboard, with JWT authentication and RBAC enforcement.

## Endpoint

`POST /graphql`

Requires JWT with `is_admin: true` claim.

## Schema

### Queries

```graphql
type Query {
  menus(storeId: ID!): [Menu!]!
  menu(id: ID!): Menu
  inventory(storeId: ID!): [InventoryItem!]!
  stockLevel(itemId: ID!, storeId: ID!): StockLevel
  orders(status: OrderStatus, dateRange: DateRange): [Order!]!
  order(id: ID!): Order
  payments(dateRange: DateRange): [Payment!]!
  payment(id: ID!): Payment
  employees(storeId: ID!): [Employee!]!
  employee(id: ID!): Employee
  kiosks(storeId: ID!): [Kiosk!]!
  kiosk(id: ID!): Kiosk
  auditLogs(dateRange: DateRange, eventType: AuditEventType): [AuditLog!]!
}
```

### Mutations

```graphql
type Mutation {
  updateMenu(id: ID!, input: MenuInput!): Menu
  updateInventory(itemId: ID!, storeId: ID!, input: InventoryInput!): StockLevel
  updateOrderStatus(id: ID!, status: OrderStatus!): Order
  createEmployee(storeId: ID!, input: EmployeeInput!): Employee
  approveRefund(paymentId: ID!): Refund
}
```

## Implementation

- **Library:** gqlgen (Go GraphQL library)
- **Auth:** JWT validation middleware extracts `tenant_id`, `role`, `is_admin`
- **RBAC:** Resolver-level permission checks against role/permission matrix
- **Data sources:** PostgreSQL queries via repository layer
