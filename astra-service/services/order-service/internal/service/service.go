// Package service implements the OrderService use cases: cart-to-order
// conversion, lifecycle transitions, and outbox event production. It is
// transport-agnostic and consumed by both the gRPC/REST server and the NATS
// event handlers.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/order-service/internal/cartclient"
	"github.com/astra-systems/astra-service/services/order-service/internal/repository"
	cartv1 "github.com/astra-systems/astra-service/proto/gen/go/cart"
	commonv1 "github.com/astra-systems/astra-service/proto/gen/go/common"
	eventsv1 "github.com/astra-systems/astra-service/proto/gen/go/events"
	orderv1 "github.com/astra-systems/astra-service/proto/gen/go/order"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	orderAggregateType = "order"
	currencyUSD        = "USD"

	EventTypeOrderCreated    = "astra.order.created.v1"
	EventTypeOrderPaid       = "astra.order.paid.v1"
	EventTypeOrderFulfilled  = "astra.order.fulfilled.v1"
	EventTypeOrderCancelled  = "astra.order.cancelled.v1"
	EventTypeOrderStatusChanged = "astra.order.status_changed.v1"

	StatusPending    = "pending"
	StatusPaid       = "paid"
	StatusFulfilled  = "fulfilled"
	StatusCancelled  = "cancelled"
	StatusRefunded   = "refunded"
)

// OrderService implements the order domain use cases.
type OrderService struct {
	repo       repository.Repository
	cartClient cartclient.Client
}

// NewOrderService returns a service backed by the supplied repository and cart
// client.
func NewOrderService(repo repository.Repository, cartClient cartclient.Client) *OrderService {
	return &OrderService{
		repo:       repo,
		cartClient: cartClient,
	}
}

// CreateOrder converts a cart into a pending order. Idempotency is enforced by
// the idempotency key scoped to the cart.
func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest, idempotencyKey string) (*orderv1.Order, error) {
	if req.CartId == "" {
		return nil, status.Error(codes.InvalidArgument, "cart_id is required")
	}

	if idempotencyKey != "" {
		existing, err := s.repo.GetOrderByIdempotencyKey(ctx, req.CartId, idempotencyKey)
		if err != nil && !errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Errorf(codes.Internal, "idempotency lookup failed: %v", err)
		}
		if existing != nil {
			return toProto(existing), nil
		}
	}

	cart, err := s.cartClient.GetCart(ctx, req.CartId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "cart not found: %v", err)
	}

	order, err := s.buildOrderFromCart(req, cart, idempotencyKey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "build order: %v", err)
	}

	eventID := uuid.New().String()
	payload, err := buildOrderCreatedEvent(order)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build event: %v", err)
	}

	if err := s.repo.CreateOrder(ctx, order, repository.OutboxEvent{
		EventID:      eventID,
		EventType:    EventTypeOrderCreated,
		Payload:      payload,
		OccurredAtMs: order.CreatedAt.UnixMilli(),
	}); err != nil {
		if errors.Is(err, repository.ErrOrderConflict) {
			return nil, status.Errorf(codes.AlreadyExists, "order already exists for cart: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "create order: %v", err)
	}

	stored, err := s.repo.GetOrder(ctx, order.OrderID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reload order: %v", err)
	}
	return toProto(stored), nil
}

// GetOrder returns a single order by ID.
func (s *OrderService) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.Order, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "order %s not found", req.OrderId)
		}
		return nil, status.Errorf(codes.Internal, "get order: %v", err)
	}
	return toProto(order), nil
}

// ListOrders returns a filtered, paginated list.
func (s *OrderService) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	page := int32(1)
	pageSize := int32(20)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
	}

	statusFilter := ""
	if req.Status != orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		statusFilter = statusFromProto(req.Status)
	}

	orders, total, err := s.repo.ListOrders(ctx, repository.ListFilter{
		StoreID:  req.StoreId,
		KioskID:  req.KioskId,
		Status:   statusFilter,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list orders: %v", err)
	}

	resp := &orderv1.ListOrdersResponse{
		Orders: make([]*orderv1.Order, 0, len(orders)),
		Pagination: &commonv1.PaginationResponse{
			Page:      page,
			PageSize:  pageSize,
			Total:     total,
			HasMore:   total > int64(page*pageSize),
		},
	}
	for _, order := range orders {
		resp.Orders = append(resp.Orders, toProto(order))
	}
	return resp, nil
}

// UpdateOrderStatus performs an explicit status transition.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.Order, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.Status == orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	newStatus := statusFromProto(req.Status)
	eventID := uuid.New().String()
	payload, err := json.Marshal(map[string]any{
		"order_id": req.OrderId,
		"status":   newStatus,
		"reason":   req.Reason,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal event: %v", err)
	}

	order, err := s.repo.UpdateOrderStatus(ctx, req.OrderId, newStatus, repository.OutboxEvent{
		EventID:      eventID,
		EventType:    EventTypeOrderStatusChanged,
		Payload:      payload,
		OccurredAtMs: time.Now().UTC().UnixMilli(),
	})
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "order %s not found", req.OrderId)
		}
		return nil, status.Errorf(codes.Internal, "update status: %v", err)
	}
	return toProto(order), nil
}

// FulfillOrder transitions a paid order to fulfilled.
func (s *OrderService) FulfillOrder(ctx context.Context, req *orderv1.FulfillOrderRequest) (*orderv1.Order, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	fulfilledAt := time.Now().UTC()
	eventID := uuid.New().String()
	payload, err := buildOrderFulfilledEvent(req.OrderId, req.FulfilledBy, fulfilledAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build event: %v", err)
	}

	order, err := s.repo.MarkFulfilled(ctx, req.OrderId, req.FulfilledBy, fulfilledAt, repository.OutboxEvent{
		EventID:      eventID,
		EventType:    EventTypeOrderFulfilled,
		Payload:      payload,
		OccurredAtMs: fulfilledAt.UnixMilli(),
	})
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "order %s not found", req.OrderId)
		}
		if errors.Is(err, repository.ErrInvalidStatus) {
			return nil, status.Errorf(codes.FailedPrecondition, "cannot fulfill order: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "fulfill order: %v", err)
	}
	return toProto(order), nil
}

// HandleCartFinalized creates an order when a cart is finalized. The cart_id
// acts as the idempotency scope, so duplicate events never create duplicate
// orders.
func (s *OrderService) HandleCartFinalized(ctx context.Context, evt *eventsv1.CartFinalized) (*orderv1.Order, error) {
	if evt.CartId == "" {
		return nil, fmt.Errorf("cart_id is required")
	}

	existing, err := s.repo.GetOrderByCartID(ctx, evt.CartId)
	if err != nil && !errors.Is(err, repository.ErrOrderNotFound) {
		return nil, fmt.Errorf("cart lookup failed: %w", err)
	}
	if existing != nil {
		return toProto(existing), nil
	}

	cart, err := s.cartClient.GetCart(ctx, evt.CartId)
	if err != nil {
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	orderID := evt.OrderId
	if orderID == "" {
		orderID = uuid.New().String()
	}

	order, err := s.buildOrderFromCart(
		&orderv1.CreateOrderRequest{CartId: evt.CartId},
		cart,
		"cart-finalized-"+evt.CartId,
	)
	if err != nil {
		return nil, fmt.Errorf("build order: %w", err)
	}
	order.OrderID = orderID
	if evt.Currency != "" {
		order.Currency = evt.Currency
	}
	if evt.FinalTotalCents > 0 {
		order.TotalCents = evt.FinalTotalCents
	}

	eventID := uuid.New().String()
	payload, err := buildOrderCreatedEvent(order)
	if err != nil {
		return nil, fmt.Errorf("build event: %w", err)
	}

	if err := s.repo.CreateOrder(ctx, order, repository.OutboxEvent{
		EventID:      eventID,
		EventType:    EventTypeOrderCreated,
		Payload:      payload,
		OccurredAtMs: order.CreatedAt.UnixMilli(),
	}); err != nil {
		if errors.Is(err, repository.ErrOrderConflict) {
			stored, err := s.repo.GetOrderByCartID(ctx, evt.CartId)
			if err != nil {
				return nil, fmt.Errorf("reload after conflict: %w", err)
			}
			return toProto(stored), nil
		}
		return nil, fmt.Errorf("create order: %w", err)
	}

	stored, err := s.repo.GetOrder(ctx, order.OrderID)
	if err != nil {
		return nil, fmt.Errorf("reload order: %w", err)
	}
	return toProto(stored), nil
}

// HandlePaymentConfirmed transitions a pending order to paid.
func (s *OrderService) HandlePaymentConfirmed(ctx context.Context, evt *eventsv1.PaymentConfirmed) error {
	if evt.OrderId == "" {
		return fmt.Errorf("order_id is required")
	}
	if evt.Status != "authorized" && evt.Status != "captured" {
		return nil
	}

	paidAt := time.Now().UTC()
	eventID := uuid.New().String()
	payload, err := buildOrderPaidEvent(evt.OrderId, evt.PaymentId, evt.AuthCode, paidAt)
	if err != nil {
		return fmt.Errorf("build event: %w", err)
	}

	_, err = s.repo.MarkPaid(ctx, evt.OrderId, paidAt, repository.OutboxEvent{
		EventID:      eventID,
		EventType:    EventTypeOrderPaid,
		Payload:      payload,
		OccurredAtMs: paidAt.UnixMilli(),
	})
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil // payment confirmed for unknown order is not a fatal error
		}
		return fmt.Errorf("mark paid: %w", err)
	}
	return nil
}

func (s *OrderService) buildOrderFromCart(req *orderv1.CreateOrderRequest, cart *cartv1.Cart, idempotencyKey string) (*repository.Order, error) {
	storeID := cart.StoreId
	if req.StoreId != "" {
		storeID = req.StoreId
	}
	kioskID := cart.KioskId
	if req.KioskId != "" {
		kioskID = req.KioskId
	}

	if storeID == "" || kioskID == "" {
		return nil, fmt.Errorf("store_id and kiosk_id are required")
	}

	orderID := uuid.New().String()
	now := time.Now().UTC()

	order := &repository.Order{
		OrderID:        orderID,
		StoreID:        storeID,
		KioskID:        kioskID,
		CartID:         cart.CartId,
		OrderNumber:    generateOrderNumber(now),
		Status:         StatusPending,
		SubtotalCents:  cart.TotalCents,
		TaxCents:       cart.TaxCents,
		DiscountCents:  cart.DiscountCents,
		TotalCents:     cart.FinalTotalCents,
		Currency:       currencyUSD,
		Items:          make([]repository.OrderItem, 0, len(cart.Lines)),
		TaxBreakdown:   map[string]string{},
		Metadata:       map[string]string{},
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	for _, line := range cart.Lines {
		modifierIDs := make([]string, 0, len(line.Modifiers))
		for _, m := range line.Modifiers {
			modifierIDs = append(modifierIDs, m.ModifierOptionId)
		}
		order.Items = append(order.Items, repository.OrderItem{
			OrderItemID:        uuid.New().String(),
			ItemID:             line.MenuItemId,
			NameSnapshot:       line.NameSnapshot,
			PriceCentsSnapshot: line.UnitPriceCentsSnapshot,
			Quantity:           line.Quantity,
			ModifierOptionIDs:  modifierIDs,
			LineTotalCents:     line.LineTotalCents,
			CreatedAt:          now,
		})
	}

	return order, nil
}

func generateOrderNumber(now time.Time) string {
	return fmt.Sprintf("A-%s", now.Format("20060102-150405"))
}

func buildOrderCreatedEvent(order *repository.Order) ([]byte, error) {
	evt := &eventsv1.OrderCreated{
		OrderId:    order.OrderID,
		CartId:     order.CartID,
		StoreId:    order.StoreID,
		KioskId:    order.KioskID,
		TotalCents: order.TotalCents,
		Currency:   order.Currency,
	}
	return marshalEnvelope(EventTypeOrderCreated, order.OrderID, evt)
}

func buildOrderPaidEvent(orderID, paymentID, authCode string, paidAt time.Time) ([]byte, error) {
	payload := map[string]any{
		"order_id":    orderID,
		"payment_id":  paymentID,
		"auth_code":   authCode,
		"paid_at":     paidAt.Format(time.RFC3339),
		"timestamp":   paidAt.Format(time.RFC3339),
	}
	return json.Marshal(payload)
}

func buildOrderFulfilledEvent(orderID, fulfilledBy string, fulfilledAt time.Time) ([]byte, error) {
	payload := map[string]any{
		"order_id":     orderID,
		"fulfilled_by": fulfilledBy,
		"fulfilled_at": fulfilledAt.Format(time.RFC3339),
		"timestamp":    fulfilledAt.Format(time.RFC3339),
	}
	return json.Marshal(payload)
}

func marshalEnvelope(eventType, aggregateID string, payload proto.Message) ([]byte, error) {
	payloadAny, err := anypb.New(payload)
	if err != nil {
		return nil, fmt.Errorf("anypb.New: %w", err)
	}
	env := &eventsv1.EventEnvelope{
		EventId:        uuid.New().String(),
		AggregateId:    aggregateID,
		AggregateType:  orderAggregateType,
		SequenceNumber: 1,
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Payload:        payloadAny,
		Metadata:       map[string]string{"event_type": eventType},
	}
	return json.Marshal(env)
}

func statusFromProto(s orderv1.OrderStatus) string {
	switch s {
	case orderv1.OrderStatus_ORDER_STATUS_PENDING:
		return StatusPending
	case orderv1.OrderStatus_ORDER_STATUS_PAID:
		return StatusPaid
	case orderv1.OrderStatus_ORDER_STATUS_FULFILLED:
		return StatusFulfilled
	case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return StatusCancelled
	case orderv1.OrderStatus_ORDER_STATUS_REFUNDED:
		return StatusRefunded
	default:
		return ""
	}
}

func statusToProto(s string) orderv1.OrderStatus {
	switch s {
	case StatusPending:
		return orderv1.OrderStatus_ORDER_STATUS_PENDING
	case StatusPaid:
		return orderv1.OrderStatus_ORDER_STATUS_PAID
	case StatusFulfilled:
		return orderv1.OrderStatus_ORDER_STATUS_FULFILLED
	case StatusCancelled:
		return orderv1.OrderStatus_ORDER_STATUS_CANCELLED
	case StatusRefunded:
		return orderv1.OrderStatus_ORDER_STATUS_REFUNDED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func toProto(order *repository.Order) *orderv1.Order {
	items := make([]*orderv1.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, &orderv1.OrderItem{
			OrderItemId:        item.OrderItemID,
			ItemId:             item.ItemID,
			NameSnapshot:       item.NameSnapshot,
			PriceCentsSnapshot: item.PriceCentsSnapshot,
			Quantity:           item.Quantity,
			ModifierOptionIds:  item.ModifierOptionIDs,
			LineTotalCents:     item.LineTotalCents,
		})
	}

	return &orderv1.Order{
		OrderId:       order.OrderID,
		StoreId:       order.StoreID,
		KioskId:       order.KioskID,
		CartId:        order.CartID,
		OrderNumber:   order.OrderNumber,
		Status:        statusToProto(order.Status),
		SubtotalCents: order.SubtotalCents,
		TaxCents:      order.TaxCents,
		DiscountCents: order.DiscountCents,
		TotalCents:    order.TotalCents,
		Items:         items,
		TaxBreakdown:  order.TaxBreakdown,
		Metadata:      order.Metadata,
		PaidAt:        formatTime(order.PaidAt),
		FulfilledAt:   formatTime(order.FulfilledAt),
		CancelledAt:   formatTime(order.CancelledAt),
		CreatedAt:     order.CreatedAt.Format(time.RFC3339Nano),
	}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// Ensure OrderService satisfies the outbox.Publisher contract indirectly by
// not using it directly. The compiler check below documents that the service
// does not depend on NATS transport details.
var _ outbox.Publisher = (*nopPublisher)(nil)

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, string, []byte) error { return nil }
