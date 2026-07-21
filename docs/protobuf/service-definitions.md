# Protobuf Service Definitions

## Common Types

**File:** `proto/proto/common.proto`

```protobuf
package astra.common.v1;

message Money {
  int64 cents = 1;
  string currency = 2;  // ISO 4217
}

message HLC {
  fixed64 wall_clock = 1;
  uint32 logical = 2;
  bytes node_id = 3;
}

message UUID {
  string value = 1;
}

message PaginationRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message PaginationResponse {
  string next_page_token = 1;
  int32 total_count = 2;
}
```

## Cart Service

**File:** `proto/proto/cart.proto`

```protobuf
package astra.cart.v1;

service CartService {
  rpc CreateCart(CreateCartRequest) returns (Cart);
  rpc GetCart(GetCartRequest) returns (Cart);
  rpc AddItem(AddItemRequest) returns (Cart);
  rpc UpdateItem(UpdateItemRequest) returns (Cart);
  rpc RemoveItem(RemoveItemRequest) returns (Cart);
  rpc FinalizeCart(FinalizeCartRequest) returns (Cart);
  rpc MergeGhostCart(MergeGhostCartRequest) returns (Cart);
}

enum CartStatus {
  CART_STATUS_UNSPECIFIED = 0;
  CART_STATUS_ACTIVE = 1;
  CART_STATUS_FINALIZED = 2;
  CART_STATUS_ABANDONED = 3;
  CART_STATUS_MERGED = 4;
  CART_STATUS_CONVERTED = 5;
}

message Cart {
  string id = 1;
  string store_id = 2;
  repeated CartLine items = 3;
  CartTotals totals = 4;
  CartStatus status = 5;
  int64 version = 6;
  HLC updated_at = 7;
  string hlc_timestamp = 8;
}
```

## Order Service

**File:** `proto/proto/order.proto`

```protobuf
package astra.order.v1;

service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (Order);
  rpc GetOrder(GetOrderRequest) returns (Order);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc UpdateOrderStatus(UpdateOrderStatusRequest) returns (Order);
  rpc FulfillOrder(FulfillOrderRequest) returns (Order);
  rpc RefundOrder(RefundOrderRequest) returns (Order);
}
```

## Payment Service

**File:** `proto/proto/payment.proto`

```protobuf
package astra.payment.v1;

service PaymentOrchestrator {
  rpc InitiatePayment(InitiatePaymentRequest) returns (PaymentIntent);
  rpc CapturePayment(CapturePaymentRequest) returns (PaymentResult);
  rpc RefundPayment(RefundPaymentRequest) returns (PaymentResult);
  rpc SettleOfflineToken(SettleOfflineTokenRequest) returns (PaymentResult);
  rpc GetPaymentStatus(GetPaymentStatusRequest) returns (PaymentStatus);
}

enum PaymentMethod {
  PAYMENT_METHOD_UNSPECIFIED = 0;
  CREDIT_CARD = 1;
  DEBIT_CARD = 2;
  CASH = 3;
  MOBILE_WALLET = 4;
  GIFT_CARD = 5;
  STORE_CREDIT = 6;
}

enum PaymentStatus {
  PAYMENT_STATUS_UNSPECIFIED = 0;
  PENDING = 1;
  AUTHORIZED = 2;
  CAPTURED = 3;
  SETTLED = 4;
  FAILED = 5;
  REFUNDED = 6;
  PARTIALLY_REFUNDED = 7;
}
```

## Sync Service

**File:** `proto/proto/sync.proto`

```protobuf
package astra.sync.v1;

service SyncService {
  rpc UploadBatch(stream SyncDelta) returns (SyncAck);
  rpc DownloadBatch(DownloadBatchRequest) returns (stream SyncDelta);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc StreamHeartbeats(stream HeartbeatRequest) returns (stream HeartbeatResponse);
}

enum SyncEventType {
  SYNC_EVENT_TYPE_UNSPECIFIED = 0;
  CART_UPDATED = 1;
  ORDER_CREATED = 2;
  PAYMENT_MADE = 3;
  INVENTORY_ADJUSTED = 4;
}
```

## Inventory Service

**File:** `proto/proto/inventory.proto`

```protobuf
package astra.inventory.v1;

service InventoryService {
  rpc GetStock(GetStockRequest) returns (StockLevel);
  rpc ReserveStock(ReserveStockRequest) returns (ReservationResult);
  rpc ReleaseStock(ReleaseStockRequest) returns (ReleaseResult);
  rpc AdjustStock(AdjustStockRequest) returns (StockLevel);
  rpc StreamStockUpdates(StreamStockRequest) returns (stream StockLevel);
}
```

## Menu Service

**File:** `proto/proto/menu.proto`

```protobuf
package astra.menu.v1;

service MenuService {
  rpc GetMenu(GetMenuRequest) returns (MenuResponse);
  rpc GetCategories(GetCategoriesRequest) returns (CategoriesResponse);
  rpc GetItem(GetItemRequest) returns (Item);
  rpc SearchItems(SearchItemsRequest) returns (SearchItemsResponse);
}
```

## Auth Service

**File:** `proto/proto/auth.proto`

```protobuf
package astra.auth.v1;

service AuthService {
  rpc BeginVerification(BeginVerificationRequest) returns (BeginVerificationResponse);
  rpc VerifyAssertion(VerifyAssertionRequest) returns (VerifyAssertionResponse);
  rpc ValidateOverrideToken(ValidateOverrideTokenRequest) returns (ValidateOverrideTokenResponse);
}
```

## Domain Events

**File:** `proto/proto/events.proto`

```protobuf
package astra.events.v1;

message EventEnvelope {
  string event_id = 1;
  string event_type = 2;
  HLC timestamp = 3;
  string aggregate_id = 4;
  string aggregate_type = 5;
  bytes payload = 6;
  map<string, string> metadata = 7;
}
```

Event types: `OrderCreated`, `ItemAddedToCart`, `PaymentInitiated`, `PaymentCompleted`, `OrderFulfilled`, `StockAdjusted`, `SyncBatchReceived`
