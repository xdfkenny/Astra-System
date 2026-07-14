import { assign, createMachine, fromPromise } from "xstate";
import type { MenuItem, Order, PaymentAuthorizationResult } from "@astra/shared-types";
import type { LaneMode } from "@astra/kiosk-state";
import { cartProxy } from "@astra/kiosk-state";
import { apiClient } from "../state/apiClient";
import { defaultLogger } from "../utils/logger";

const log = defaultLogger.child("kioskMachine");

export type KioskWorkflowStage =
  | "ATTRACT"
  | "MENU"
  | "ITEM_DETAIL"
  | "CART"
  | "PAYMENT"
  | "PROCESSING"
  | "RECEIPT"
  | "ADMIN";

export interface KioskContext {
  readonly sessionId: string | null;
  readonly laneMode: LaneMode;
  readonly selectedItem: MenuItem | null;
  readonly cartHasItems: boolean;
  readonly returnTo: "MENU" | "CART" | null;
  readonly paymentResult: PaymentAuthorizationResult | null;
  readonly order: Order | null;
  readonly errorMessage: string | null;
  readonly apiStatus: "online" | "offline" | "degraded" | "unknown";
  readonly isOfflineMode: boolean;
  readonly cartId: string;
}

export type KioskEvent =
  | { type: "START_SESSION"; sessionId: string; laneMode?: LaneMode }
  | { type: "TAP_START"; sessionId: string; laneMode?: LaneMode }
  | { type: "SELECT_ITEM"; item: MenuItem }
  | { type: "CLOSE_ITEM_DETAIL" }
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
  | { type: "RETURN_TO_ATTRACT" }
  | { type: "OPEN_ADMIN" }
  | { type: "CLOSE_ADMIN" }
  | { type: "API_ERROR"; message: string }
  | { type: "NETWORK_OFFLINE" }
  | { type: "NETWORK_ONLINE" };

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

async function finalizeOrder(input: {
  sessionId: string;
  paymentResult: PaymentAuthorizationResult | null;
  cartId: string;
}): Promise<Order> {
  if (!input.paymentResult) {
    throw new Error("Cannot finalize order without a payment result.");
  }

  log.info("Finalizing order", {
    sessionId: input.sessionId,
    cartId: input.cartId,
    authorizationId: input.paymentResult.authorizationId,
  });

  const orderResponse = await apiClient.createOrder(
    input.cartId,
    input.paymentResult.authorizationId,
  );

  return orderResponse;
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
        apiStatus: "unknown",
        isOfflineMode: false,
        cartId: cartProxy.cartId,
      },
    states: {
       ATTRACT: {
        on: {
          START_SESSION: {
            target: "MENU",
            actions: ["assignSession"],
          },
          TAP_START: {
            target: "MENU",
            actions: ["assignSession"],
          },
          OPEN_ADMIN: {
            target: "ADMIN",
          },
          NETWORK_OFFLINE: {
            actions: ["setOfflineMode"],
          },
          NETWORK_ONLINE: {
            actions: ["setOnlineMode"],
          },
        },
      },
       MENU: {
         on: {
           SELECT_ITEM: {
             target: "ITEM_DETAIL",
             actions: ["assignSelectedItem"],
           },
           GO_TO_CART: {
             guard: "cartHasItems",
             target: "CART",
           },
           CART_UPDATED: {
             actions: ["assignCartStatus"],
           },
           RETURN_TO_ATTRACT: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           OPEN_ADMIN: {
             target: "ADMIN",
           },
           NETWORK_OFFLINE: {
             actions: ["setOfflineMode"],
           },
           NETWORK_ONLINE: {
             actions: ["setOnlineMode"],
           },
         },
       },
       ITEM_DETAIL: {
         on: {
           CLOSE_ITEM_DETAIL: {
             target: "MENU",
             actions: ["clearSelectedItem"],
           },
           ADD_TO_CART: {
             target: "MENU",
             actions: ["clearSelectedItem", "markCartHasItems"],
           },
           RETURN_TO_ATTRACT: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           OPEN_ADMIN: {
             target: "ADMIN",
           },
           NETWORK_OFFLINE: {
             actions: ["setOfflineMode"],
           },
           NETWORK_ONLINE: {
             actions: ["setOnlineMode"],
           },
         },
       },
       CART: {
         on: {
           BACK_TO_MENU: {
             target: "MENU",
           },
           PROCEED_TO_PAYMENT: {
             guard: "cartHasItems",
             target: "PAYMENT",
           },
           RETURN_TO_ATTRACT: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           OPEN_ADMIN: {
             target: "ADMIN",
           },
           NETWORK_OFFLINE: {
             actions: ["setOfflineMode"],
           },
           NETWORK_ONLINE: {
             actions: ["setOnlineMode"],
           },
         },
       },
       PAYMENT: {
         entry: ["clearError"],
         on: {
           PAYMENT_AUTHORIZED: {
             guard: "paymentApproved",
             target: "PROCESSING",
             actions: ["assignPaymentResult"],
           },
           PAYMENT_DECLINED: {
             guard: "paymentDeclined",
             target: "CART",
             actions: ["setDeclineError"],
           },
           PAYMENT_FAILED: {
             target: "CART",
             actions: ["setErrorMessage"],
           },
           CANCEL_PAYMENT: {
             target: "CART",
           },
           RETURN_TO_ATTRACT: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           OPEN_ADMIN: {
             target: "ADMIN",
           },
           NETWORK_OFFLINE: {
             actions: ["setOfflineMode"],
           },
           NETWORK_ONLINE: {
             actions: ["setOnlineMode"],
           },
         },
       },
        PROCESSING: {
          invoke: {
            src: "finalizeOrder",
            input: ({ context }) => ({
              sessionId: context.sessionId ?? "anonymous",
              paymentResult: context.paymentResult,
              cartId: context.cartId,
            }),
            onDone: {
             target: "RECEIPT",
             actions: assign({ order: ({ event }) => event.output as Order }),
           },
           onError: {
             target: "PAYMENT",
             actions: [
               assign({
                 errorMessage: ({ event }) =>
                   event.error instanceof Error ? event.error.message : "Order finalization failed.",
               }),
                () => {
                  log.error("Order finalization failed", undefined, {
                    sessionId: undefined,
                  });
                },
             ],
           },
         },
         on: {
           RETURN_TO_ATTRACT: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           NETWORK_OFFLINE: {
             actions: ["setOfflineMode"],
           },
           NETWORK_ONLINE: {
             actions: ["setOnlineMode"],
           },
         },
       },
       RECEIPT: {
         on: {
           RECEIPT_ACKNOWLEDGED: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           RETURN_TO_ATTRACT: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           OPEN_ADMIN: {
             target: "ADMIN",
           },
           NETWORK_OFFLINE: {
             actions: ["setOfflineMode"],
           },
           NETWORK_ONLINE: {
             actions: ["setOnlineMode"],
           },
         },
       },
        ADMIN: {
         on: {
           CLOSE_ADMIN: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           RETURN_TO_ATTRACT: {
             target: "ATTRACT",
             actions: ["resetContext"],
           },
           NETWORK_OFFLINE: {
             actions: ["setOfflineMode"],
           },
           NETWORK_ONLINE: {
             actions: ["setOnlineMode"],
           },
         },
       },
    },
  },
  {
    actions: {
      assignSession: assign(({ event }) => {
        if (event.type === "START_SESSION" || event.type === "TAP_START") {
          return {
            sessionId: event.sessionId,
            laneMode: event.laneMode ?? "full",
            errorMessage: null,
            cartId: cartProxy.cartId,
            returnTo: null,
          };
        }
        return {};
      }),
      assignSelectedItem: assign(({ event }) => ({
        selectedItem: event.type === "SELECT_ITEM" ? event.item : null,
      })),
      clearSelectedItem: assign({ selectedItem: null }),
      assignCartStatus: assign(({ event }) => ({
        cartHasItems: event.type === "CART_UPDATED" ? event.cartHasItems : false,
      })),
      markCartHasItems: assign({ cartHasItems: true }),
      assignPaymentResult: assign(({ event }) => ({
        paymentResult: event.type === "PAYMENT_AUTHORIZED" ? event.result : null,
      })),
      clearPaymentResult: assign({ paymentResult: null }),
      setPaymentError: assign(({ event }) => ({
        errorMessage:
          event.type === "PAYMENT_DECLINED"
            ? event.result.declineReason ?? "Payment declined. Try another method."
            : event.type === "PAYMENT_FAILED"
            ? event.message
            : null,
      })),
      setErrorMessage: assign(({ event }) => ({
        errorMessage: event.type === "PAYMENT_FAILED" ? event.message : "An unexpected error occurred.",
      })),
      setDeclineError: assign(({ event }) => ({
        errorMessage:
          event.type === "PAYMENT_DECLINED"
            ? event.result.declineReason ?? "Payment declined. Try another method."
            : null,
      })),
      setOnlineMode: assign({
        apiStatus: "online",
        isOfflineMode: false,
      }),
      setOfflineMode: assign({
        apiStatus: "offline",
        isOfflineMode: true,
      }),
       assignCartId: assign({
         cartId: cartProxy.cartId,
       }),
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
        cartId: cartProxy.cartId,
      })),
    },
    guards: {
      cartHasItems: ({ context }) => context.cartHasItems,
      paymentApproved: ({ event }) =>
        event.type === "PAYMENT_AUTHORIZED" && isApprovedResult(event.result),
      paymentDeclined: ({ event }) =>
        event.type === "PAYMENT_DECLINED" && isDeclinedResult(event.result),
    },
    actors: {
      finalizeOrder: fromPromise<
        Order,
        { sessionId: string; paymentResult: PaymentAuthorizationResult | null; cartId: string }
      >(async ({ input }) => finalizeOrder(input)),
    },
  },
);

