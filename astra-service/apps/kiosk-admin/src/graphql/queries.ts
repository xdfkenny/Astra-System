import { gql } from "@apollo/client";

const CORE_LOCATION_FIELDS = gql`
  fragment CoreLocationFields on Location {
    locationId
    tenantId
    slug
    name
    address
    timezone
    currency
    taxRate
    createdAt
    updatedAt
    deletedAt
  }
`;

const CORE_LANE_FIELDS = gql`
  fragment CoreLaneFields on Lane {
    laneId
    locationId
    displayName
    laneNumber
    isActive
    createdAt
    updatedAt
    deletedAt
  }
`;

export const DASHBOARD_KPIS = gql`
  query DashboardKpis {
    dashboardKpis {
      totalRevenueCents
      orderCount
      activeKiosks
      alerts
      revenueTrend
      orderTrend
    }
  }
`;

export const LIST_LOCATIONS = gql`
  ${CORE_LOCATION_FIELDS}
  ${CORE_LANE_FIELDS}
  query ListLocations {
    locations {
      items {
        ...CoreLocationFields
        lanes {
          ...CoreLaneFields
        }
      }
      totalCount
    }
  }
`;

export const LIST_LANES = gql`
  ${CORE_LANE_FIELDS}
  query ListLanes {
    lanes {
      items {
        ...CoreLaneFields
      }
      totalCount
    }
  }
`;

export const LIST_KIOSK_HEALTH = gql`
  query ListKioskHealth {
    kiosks {
      items {
        kioskId
        displayName
        syncStatus
        lastSeenAt
        isLeader
      }
      totalCount
    }
  }
`;

export const LIST_MENU = gql`
  query ListMenu {
    menu {
      categories {
        categoryId
        storeId
        parentId
        name
        description
        displayOrder
        imageUrl
        isActive
        items {
          itemId
          storeId
          categoryId
          name
          description
          priceCents
          costCents
          plu
          barcode
          sku
          imageUrl
          taxCategory
          isWeightBased
          weightUnit
          isActive
        }
      }
      modifierGroups {
        modifierGroupId
        storeId
        name
        description
        minSelect
        maxSelect
        displayOrder
        isActive
        options {
          modifierOptionId
          modifierGroupId
          name
          priceDeltaCents
          isDefault
          displayOrder
          isActive
        }
      }
    }
  }
`;

export const LIST_INVENTORY = gql`
  query ListInventory {
    inventory {
      items {
        inventoryId
        storeId
        itemId
        itemName
        itemSku
        quantityAvailable
        quantityReserved
        quantityOnOrder
        reorderPoint
        reorderQuantity
        location
        lastCountedAt
        updatedAt
      }
      totalCount
    }
  }
`;

export const LIST_ORDERS = gql`
  query ListOrders($limit: Int = 50, $offset: Int = 0) {
    orders(limit: $limit, offset: $offset) {
      items {
        orderId
        storeId
        kioskId
        cartId
        orderNumber
        status
        subtotalCents
        taxCents
        discountCents
        totalCents
        paidAt
        fulfilledAt
        createdAt
        kioskDisplayName
      }
      totalCount
    }
  }
`;

export const LIST_PAYMENTS_AND_REFUNDS = gql`
  query ListPaymentsAndRefunds($limit: Int = 50, $offset: Int = 0) {
    payments(limit: $limit, offset: $offset) {
      items {
        paymentId
        orderId
        kioskId
        amountCents
        currency
        method
        status
        cardBrand
        cardLastFour
        declineReason
        createdAt
        orderNumber
      }
      totalCount
    }
    refunds(limit: $limit, offset: $offset) {
      items {
        refundId
        paymentId
        orderId
        kioskId
        amountCents
        currency
        reason
        status
        processedBy
        createdAt
        paymentAmountCents
      }
      totalCount
    }
  }
`;

export const LIST_EMPLOYEES_AND_ROLES = gql`
  query ListEmployeesAndRoles {
    employees {
      items {
        employeeId
        storeId
        name
        email
        role
        isActive
        lastLoginAt
        createdAt
        roleName
      }
      totalCount
    }
    roles {
      items {
        roleId
        tenantId
        name
        description
        isSystem
        createdAt
        updatedAt
      }
      totalCount
    }
  }
`;

export const LIST_AUDIT_LOGS = gql`
  query ListAuditLogs($limit: Int = 100, $offset: Int = 0) {
    auditLogs(limit: $limit, offset: $offset) {
      items {
        auditId
        storeId
        tenantId
        laneId
        kioskId
        employeeId
        userId
        eventType
        entityType
        entityId
        payloadJson
        previousHash
        currentHash
        createdAt
        actorName
      }
      totalCount
    }
  }
`;
