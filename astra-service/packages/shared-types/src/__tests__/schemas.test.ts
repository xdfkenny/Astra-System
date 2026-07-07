import { describe, expect, it } from "vitest";
import {
  CartSchema,
  CategorySchema,
  EmployeeSchema,
  ItemSchema,
  LocationSchema,
  MenuResponseSchema,
  ModifierGroupSchema,
  OrderSchema,
  PaymentSchema,
  TenantSchema,
  CheckoutRequestSchema,
  AddItemRequestSchema,
  CreateCartRequestSchema,
  PaymentResultSchema,
} from "../schemas";
import { generateId } from "../ids";

const now = new Date().toISOString();
const hmac = "a".repeat(64);

const makeTenant = () =>
  ({
    tenantId: generateId(),
    slug: "astra-demo",
    name: "Astra Demo Tenant",
    billingEmail: "billing@astra-demo.internal",
    plan: "standard" as const,
    isActive: true,
    createdAt: now,
    updatedAt: now,
    deletedAt: null,
  });

const makeLocation = () =>
  ({
    locationId: generateId(),
    tenantId: generateId(),
    slug: "miami-brickell",
    name: "Astra Miami Brickell",
    address: "1000 Brickell Ave, Miami, FL 33131",
    timezone: "America/New_York",
    currency: "USD",
    taxRate: 0.07,
    createdAt: now,
    updatedAt: now,
    deletedAt: null,
  });

const makeCategory = () =>
  ({
    categoryId: generateId(),
    storeId: generateId(),
    parentId: null,
    name: "Produce",
    description: "Fresh fruits and vegetables",
    displayOrder: 1,
    imageUrl: null,
    blurhash: null,
    isActive: true,
    createdAt: now,
    updatedAt: now,
    deletedAt: null,
  });

const makeItem = () =>
  ({
    itemId: generateId(),
    storeId: generateId(),
    categoryId: generateId(),
    name: "Banana",
    description: "Organic banana",
    priceCents: 99,
    costCents: 50,
    plu: null,
    barcode: null,
    sku: "BAN-001",
    imageUrl: null,
    blurhash: null,
    taxCategory: "standard" as const,
    isWeightBased: false,
    weightUnit: null,
    isActive: true,
    metadata: null,
    createdAt: now,
    updatedAt: now,
    deletedAt: null,
  });

const makeModifierGroup = () =>
  ({
    modifierGroupId: generateId(),
    storeId: generateId(),
    name: "Milk options",
    description: null,
    minSelect: 0,
    maxSelect: 1,
    displayOrder: 0,
    isActive: true,
    createdAt: now,
    updatedAt: now,
    deletedAt: null,
  });

const makeCartLine = () =>
  ({
    lineId: generateId(),
    cartId: generateId(),
    menuItemId: generateId(),
    nameSnapshot: "Banana",
    unitPriceCentsSnapshot: 99,
    quantity: 3,
    modifiers: [],
    addedAtMs: Date.now(),
  });

const makeCart = () =>
  ({
    cartId: generateId(),
    storeId: generateId(),
    kioskId: generateId(),
    sessionId: generateId(),
    customerPhone: null,
    status: "active" as const,
    finalized: false,
    version: 1,
    totalCents: 297,
    taxCents: 21,
    discountCents: 0,
    finalTotalCents: 318,
    itemsJson: [makeCartLine()],
    reservedInventory: false,
    expiresAt: now,
    createdAt: now,
    updatedAt: now,
    createdAtMs: Date.now(),
    updatedAtMs: Date.now(),
  });

const makeOrderItem = () =>
  ({
    orderItemId: generateId(),
    orderId: generateId(),
    itemId: generateId(),
    nameSnapshot: "Banana",
    priceCentsSnapshot: 99,
    quantity: 3,
    modifiersJson: [],
    lineTotalCents: 297,
    createdAt: now,
  });

const makeOrder = () =>
  ({
    orderId: generateId(),
    storeId: generateId(),
    kioskId: generateId(),
    cartId: generateId(),
    orderNumber: "A-001",
    status: "pending" as const,
    subtotalCents: 297,
    taxCents: 21,
    discountCents: 0,
    totalCents: 318,
    itemsJson: [makeOrderItem()],
    taxBreakdownJson: null,
    metadata: null,
    paidAt: null,
    fulfilledAt: null,
    cancelledAt: null,
    createdAt: now,
  });

const makePayment = () =>
  ({
    paymentId: generateId(),
    orderId: generateId(),
    kioskId: generateId(),
    idempotencyKey: generateId(),
    amountCents: 318,
    currency: "USD",
    method: "credit_debit" as const,
    status: "captured" as const,
    verifoneToken: null,
    verifoneAuthCode: "123456",
    cardBrand: "visa",
    cardLastFour: "4242",
    declineReason: null,
    receiptText: null,
    isOfflineToken: false,
    offlineTokenHmac: null,
    syncedAt: null,
    createdAt: now,
    updatedAt: now,
  });

describe("domain schemas", () => {
  it("parses a valid tenant", () => {
    expect(TenantSchema.safeParse(makeTenant()).success).toBe(true);
  });

  it("rejects an invalid billing email", () => {
    const tenant = { ...makeTenant(), billingEmail: "not-an-email" };
    expect(TenantSchema.safeParse(tenant).success).toBe(false);
  });

  it("parses a valid location", () => {
    expect(LocationSchema.safeParse(makeLocation()).success).toBe(true);
  });

  it("rejects a tax rate outside [0,1]", () => {
    const location = { ...makeLocation(), taxRate: 1.5 };
    expect(LocationSchema.safeParse(location).success).toBe(false);
  });

  it("parses a valid category", () => {
    expect(CategorySchema.safeParse(makeCategory()).success).toBe(true);
  });

  it("parses a valid item", () => {
    expect(ItemSchema.safeParse(makeItem()).success).toBe(true);
  });

  it("rejects negative price cents", () => {
    const item = { ...makeItem(), priceCents: -1 };
    expect(ItemSchema.safeParse(item).success).toBe(false);
  });

  it("parses a valid modifier group", () => {
    expect(ModifierGroupSchema.safeParse(makeModifierGroup()).success).toBe(true);
  });

  it("rejects modifier group with minSelect > maxSelect", () => {
    const group = { ...makeModifierGroup(), minSelect: 2, maxSelect: 1 };
    expect(ModifierGroupSchema.safeParse(group).success).toBe(false);
  });

  it("parses a valid cart", () => {
    expect(CartSchema.safeParse(makeCart()).success).toBe(true);
  });

  it("rejects finalized cart with no line items", () => {
    const cart = { ...makeCart(), status: "finalized" as const, finalized: true, itemsJson: [] };
    expect(CartSchema.safeParse(cart).success).toBe(false);
  });

  it("parses a valid order", () => {
    expect(OrderSchema.safeParse(makeOrder()).success).toBe(true);
  });

  it("parses a valid payment", () => {
    expect(PaymentSchema.safeParse(makePayment()).success).toBe(true);
  });

  it("parses a valid employee", () => {
    const employee = {
      employeeId: generateId(),
      storeId: generateId(),
      name: "John Cashier",
      email: "john@astra.internal",
      role: "cashier" as const,
      biometricHash: hmac,
      webauthnCredentialId: null,
      webauthnPublicKey: null,
      isActive: true,
      lastLoginAt: null,
      createdAt: now,
      updatedAt: now,
      deletedAt: null,
    };
    expect(EmployeeSchema.safeParse(employee).success).toBe(true);
  });

  it("rejects employee with malformed email", () => {
    const employee = {
      employeeId: generateId(),
      storeId: generateId(),
      name: "John Cashier",
      email: "not-an-email",
      role: "cashier" as const,
      biometricHash: hmac,
      webauthnCredentialId: null,
      webauthnPublicKey: null,
      isActive: true,
      lastLoginAt: null,
      createdAt: now,
      updatedAt: now,
      deletedAt: null,
    };
    expect(EmployeeSchema.safeParse(employee).success).toBe(false);
  });
});

describe("api schemas", () => {
  it("parses a create cart request", () => {
    const request = {
      storeId: generateId(),
      kioskId: generateId(),
      sessionId: generateId(),
    };
    expect(CreateCartRequestSchema.safeParse(request).success).toBe(true);
  });

  it("parses an add item request", () => {
    const request = {
      cartId: generateId(),
      menuItemId: generateId(),
      nameSnapshot: "Banana",
      unitPriceCentsSnapshot: 99,
      quantity: 2,
    };
    expect(AddItemRequestSchema.safeParse(request).success).toBe(true);
  });

  it("rejects add item request with zero quantity", () => {
    const request = {
      cartId: generateId(),
      menuItemId: generateId(),
      nameSnapshot: "Banana",
      unitPriceCentsSnapshot: 99,
      quantity: 0,
    };
    expect(AddItemRequestSchema.safeParse(request).success).toBe(false);
  });

  it("parses a checkout request", () => {
    const request = {
      cartId: generateId(),
      method: "credit_debit" as const,
    };
    expect(CheckoutRequestSchema.safeParse(request).success).toBe(true);
  });

  it("parses a payment result", () => {
    const result = {
      paymentId: generateId(),
      orderId: generateId(),
      cartId: generateId(),
      amountCents: 318,
      currency: "USD",
      method: "credit_debit" as const,
      status: "captured" as const,
      authorization: {
        authorizationId: generateId(),
        status: "captured" as const,
        method: "credit_debit" as const,
        amountCents: 318,
      },
    };
    expect(PaymentResultSchema.safeParse(result).success).toBe(true);
  });

  it("parses a menu response", () => {
    const response = {
      storeId: generateId(),
      currency: "USD",
      taxRate: 0.07,
      categories: [makeCategory()],
      items: [],
    };
    expect(MenuResponseSchema.safeParse(response).success).toBe(true);
  });
});
