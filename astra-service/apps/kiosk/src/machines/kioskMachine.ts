import { assign, createMachine, fromPromise } from "xstate";
import type { MenuItem, Order, PaymentAuthorizationResult } from "@astra/shared-types";
import type { LaneMode } from "@astra/kiosk-state";

/**
 * Unified kiosk workflow state machine (XState v5).
 *
 * The kiosk is a physically-embedded, unattended device. This machine is the
 * single source of truth for which high-level stage the customer is in. It
 * prevents illegal jumps (e.g. straight to receipt without payment), enforces
 * the idle-reclaim flow, and centralizes the async payment finalization actor.
 */

export type KioskWorkflowStage =
  | "ATTRACT"
  | "IDLE_TIMEOUT"
  | "MENU_BROWSE"
  | "ITEM_MODAL"
  | "CART_REVIEW"
  | "PAYMENT_AUTH"
  | "PROCESSING"
  | "RECEIPT"
  | "RESET";

export interface KioskContext {
  readonly sessionId: string | null;
  readonly laneMode: LaneMode;
  readonly selectedItem: MenuItem | null;
  readonly cartHasItems: boolean;
  readonly returnTo: "MENU_BROWSE" | "CART_REVIEW" | null;
  readonly paymentResult: PaymentAuthorizationResult | null;
  readonly order: Order | null;
  readonly errorMessage: string | null;
}

export type KioskEvent =
  | { type: "START_SESSION"; sessionId: string; laneMode?: LaneMode }
  | { type: "IDLE_TIMEOUT" }
  | { type: "CONTINUE_SESSION" }
  | { type: "RESET_SESSION" }
  | { type: "SELECT_ITEM"; item: MenuItem }
  | { type: "CLOSE_ITEM_MODAL" }
  | { type: "ADD_TO_CART" }
  | { type: "CART_UPDATED"; cartHasItems: boolean }
  | { type: "GO_TO_CART" }
  | { type: "BACK_TO_MENU" }
  | { type: "PROCEED_TO_PAYMENT" }
  | { type: "PAYMENT_AUTHORIZED"; result: PaymentAuthorizationResult }
  | { type: "PAYMENT_DECLINED"; result: PaymentAuthorizationResult }
  | { type: "PAYMENT_FAILED"; message: string }
  | { type: "CANCEL_PAYMENT" }
  | { type: "ORDER_FINALIZED"; order: Order }
  | { type: "RECEIPT_ACKNOWLEDGED" }
  | { type: "RETURN_TO_ATTRACT" };

const APPROVED_STATUSES: readonly PaymentAuthorizationResult["status"][] = [
  "authorized",
  "captured",
  "queued_offline",
];

function isApprovedResult(result: PaymentAuthorizationResult): boolean {
  return APPROVED_STATUSES.includes(result.status);
}

function isDeclinedResult(result: PaymentAuthorizationResult): boolean {
  return result.status === "declined";
}

/**
 * Simulates the backend order finalization that happens after the terminal
 * reports a successful authorization. In production this would POST to the
 * local astra-syncd outbox (or the cloud order-service when online).
 */
async function finalizeOrder(input: {
  sessionId: string;
  paymentResult: PaymentAuthorizationResult | null;
}): Promise<Order> {
  if (!input.paymentResult) {
    throw new Error("Cannot finalize order without a payment result.");
  }
  await new Promise((resolve) => {
    setTimeout(resolve, 800);
  });
  return {
    orderId: crypto.randomUUID(),
    storeId: crypto.randomUUID(),
    kioskId: "kiosk-local",
    cartId: crypto.randomUUID(),
    orderNumber: `${Math.floor(Math.random() * 8999) + 1000}`,
    status: "paid",
    subtotalCents: input.paymentResult.amountCents,
    taxCents: 0,
    discountCents: 0,
    totalCents: input.paymentResult.amountCents,
    itemsJson: [],
    taxBreakdownJson: null,
    metadata: { paymentAuthorizationId: input.paymentResult.authorizationId },
    paidAt: new Date().toISOString(),
    fulfilledAt: null,
    cancelledAt: null,
    createdAt: new Date().toISOString(),
  };
}

export const kioskMachine = createMachine(
  {
    id: "kiosk",
    types: {
      context: {} as KioskContext,
      events: {} as KioskEvent,
    },
    initial: "ATTRACT",
    context: {
      sessionId: null,
      laneMode: "full",
      selectedItem: null,
      cartHasItems: false,
      returnTo: null,
      paymentResult: null,
      order: null,
      errorMessage: null,
    },
    states: {
      ATTRACT: {
        on: {
          START_SESSION: {
            target: "MENU_BROWSE",
            actions: ["assignSession"],
          },
        },
      },
      MENU_BROWSE: {
        on: {
          SELECT_ITEM: {
            target: "ITEM_MODAL",
            actions: ["assignSelectedItem"],
          },
          GO_TO_CART: {
            guard: "cartHasItems",
            target: "CART_REVIEW",
          },
          CART_UPDATED: {
            actions: ["assignCartStatus"],
          },
          IDLE_TIMEOUT: {
            target: "IDLE_TIMEOUT",
            actions: ["assignReturnToMenu"],
          },
        },
      },
      ITEM_MODAL: {
        on: {
          CLOSE_ITEM_MODAL: {
            target: "MENU_BROWSE",
            actions: ["clearSelectedItem"],
          },
          ADD_TO_CART: {
            target: "MENU_BROWSE",
            actions: ["clearSelectedItem", "markCartHasItems"],
          },
        },
      },
      CART_REVIEW: {
        on: {
          BACK_TO_MENU: {
            target: "MENU_BROWSE",
          },
          PROCEED_TO_PAYMENT: {
            guard: "cartHasItems",
            target: "PAYMENT_AUTH",
          },
          IDLE_TIMEOUT: {
            target: "IDLE_TIMEOUT",
            actions: ["assignReturnToCart"],
          },
        },
      },
      PAYMENT_AUTH: {
        entry: ["clearError"],
        on: {
          PAYMENT_AUTHORIZED: {
            guard: "paymentApproved",
            target: "PROCESSING",
            actions: ["assignPaymentResult"],
          },
          PAYMENT_DECLINED: {
            guard: "paymentDeclined",
            target: "CART_REVIEW",
            actions: ["setDeclineError"],
          },
          PAYMENT_FAILED: {
            target: "CART_REVIEW",
            actions: ["setErrorMessage"],
          },
          CANCEL_PAYMENT: {
            target: "CART_REVIEW",
          },
        },
      },
      PROCESSING: {
        invoke: {
          src: "finalizeOrder",
          input: ({ context }) => ({
            sessionId: context.sessionId ?? "anonymous",
            paymentResult: context.paymentResult,
          }),
          onDone: {
            target: "RECEIPT",
            actions: assign({ order: ({ event }) => event.output as Order }),
          },
          onError: {
            target: "PAYMENT_AUTH",
            actions: assign({
              errorMessage: ({ event }) =>
                event.error instanceof Error ? event.error.message : "Order finalization failed.",
            }),
          },
        },
      },
      RECEIPT: {
        after: {
          8000: { target: "RESET" },
        },
        on: {
          RECEIPT_ACKNOWLEDGED: { target: "RESET" },
        },
      },
      RESET: {
        entry: ["resetContext"],
        always: { target: "ATTRACT" },
      },
      IDLE_TIMEOUT: {
        after: {
          10000: { target: "RESET" },
        },
        on: {
          CONTINUE_SESSION: [
            {
              guard: "returnedToCart",
              target: "CART_REVIEW",
            },
            {
              target: "MENU_BROWSE",
            },
          ],
          RESET_SESSION: { target: "RESET" },
        },
      },
    },
  },
  {
    actions: {
      assignSession: assign(({ event }) => ({
        sessionId: event.type === "START_SESSION" ? event.sessionId : null,
        laneMode: event.type === "START_SESSION" ? (event.laneMode ?? "full") : "full",
        errorMessage: null,
      })),
      assignSelectedItem: assign(({ event }) => ({
        selectedItem: event.type === "SELECT_ITEM" ? event.item : null,
      })),
      clearSelectedItem: assign({ selectedItem: null }),
      assignCartStatus: assign(({ event }) => ({
        cartHasItems: event.type === "CART_UPDATED" ? event.cartHasItems : false,
      })),
      markCartHasItems: assign({ cartHasItems: true }),
      assignReturnToMenu: assign({ returnTo: "MENU_BROWSE" }),
      assignReturnToCart: assign({ returnTo: "CART_REVIEW" }),
      assignPaymentResult: assign(({ event }) => ({
        paymentResult: event.type === "PAYMENT_AUTHORIZED" ? event.result : null,
      })),
      setDeclineError: assign(({ event }) => ({
        errorMessage:
          event.type === "PAYMENT_DECLINED"
            ? event.result.declineReason ?? "Payment declined. Try another method."
            : null,
      })),
      setErrorMessage: assign(({ event }) => ({
        errorMessage: event.type === "PAYMENT_FAILED" ? event.message : "An unexpected error occurred.",
      })),
      clearError: assign({ errorMessage: null }),
      resetContext: assign(() => ({
        sessionId: null,
        laneMode: "full",
        selectedItem: null,
        cartHasItems: false,
        returnTo: null,
        paymentResult: null,
        order: null,
        errorMessage: null,
      })),
    },
    guards: {
      cartHasItems: ({ context }) => context.cartHasItems,
      paymentApproved: ({ event }) =>
        event.type === "PAYMENT_AUTHORIZED" && isApprovedResult(event.result),
      paymentDeclined: ({ event }) =>
        event.type === "PAYMENT_DECLINED" && isDeclinedResult(event.result),
      returnedToCart: ({ context }) => context.returnTo === "CART_REVIEW",
    },
    actors: {
      finalizeOrder: fromPromise<
        Order,
        { sessionId: string; paymentResult: PaymentAuthorizationResult | null }
      >(async ({ input }) => finalizeOrder(input)),
    },
  },
);
