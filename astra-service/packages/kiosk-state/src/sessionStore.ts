import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";

/**
 * Global kiosk session state machine (Zustand).
 *
 * WHY Zustand for this and Valtio for cart: session/workflow state is
 * discrete and event-driven (finite state machine transitions triggered by
 * touch/timer/payment events) — Zustand's selector-based subscriptions avoid
 * re-rendering the whole tree on every tick. Cart contents are a deeply
 * nested, frequently-mutated object graph shared with the CRDT/WASM layer,
 * where Valtio's proxy mutation model maps 1:1 onto CRDT patch semantics.
 */
export type WorkflowStage =
  | "attract"
  | "menu"
  | "cart_review"
  | "payment_auth"
  | "receipt";

export type LaneMode = "express" | "full";

export interface NetworkStatus {
  readonly online: boolean;
  readonly syncLagMs: number;
  readonly meshPeerCount: number;
  readonly isLeader: boolean;
}

interface SessionState {
  stage: WorkflowStage;
  laneMode: LaneMode;
  sessionId: string | null;
  lastInteractionAtMs: number;
  network: NetworkStatus;
  silentAssistArmed: boolean;

  startSession: (sessionId: string) => void;
  endSession: () => void;
  goToStage: (stage: WorkflowStage) => void;
  recordInteraction: () => void;
  setLaneMode: (mode: LaneMode) => void;
  setNetworkStatus: (status: Partial<NetworkStatus>) => void;
  armSilentAssist: (armed: boolean) => void;
}

/** Valid forward/backward transitions — prevents impossible UI states (e.g. skipping payment auth). */
const ALLOWED_TRANSITIONS: Record<WorkflowStage, ReadonlySet<WorkflowStage>> = {
  attract: new Set(["menu"]),
  menu: new Set(["cart_review", "attract"]),
  cart_review: new Set(["menu", "payment_auth", "attract"]),
  payment_auth: new Set(["receipt", "cart_review"]), // back-nav allowed if auth fails/cancels
  receipt: new Set(["attract"]),
};

export const useSessionStore = create<SessionState>()(
  subscribeWithSelector((set, get) => ({
    stage: "attract",
    laneMode: "full",
    sessionId: null,
    lastInteractionAtMs: Date.now(),
    network: { online: true, syncLagMs: 0, meshPeerCount: 0, isLeader: false },
    silentAssistArmed: false,

    startSession: (sessionId) => {
      set({ sessionId, stage: "menu", lastInteractionAtMs: Date.now() });
    },

    endSession: () => {
      set({
        sessionId: null,
        stage: "attract",
        silentAssistArmed: false,
        lastInteractionAtMs: Date.now(),
      });
    },

    goToStage: (stage) => {
      const current = get().stage;
      if (!ALLOWED_TRANSITIONS[current].has(stage)) {
        // Fail loud in dev, fail safe (no-op) in prod — an invalid transition
        // attempt indicates a bug in a micro-frontend, not a user action to punish.
        if (import.meta.env.DEV) {
          throw new Error(`Illegal workflow transition: ${current} -> ${stage}`);
        }
        return;
      }
      set({ stage, lastInteractionAtMs: Date.now(), silentAssistArmed: false });
    },

    recordInteraction: () => {
      set({ lastInteractionAtMs: Date.now(), silentAssistArmed: false });
    },

    setLaneMode: (laneMode) => {
      set({ laneMode });
    },

    setNetworkStatus: (status) => {
      set((prev) => ({ network: { ...prev.network, ...status } }));
    },

    armSilentAssist: (silentAssistArmed) => {
      set({ silentAssistArmed });
    },
  })),
);

/** Idle threshold for the Attract Loop to reclaim an abandoned session. */
export const SESSION_IDLE_TIMEOUT_MS = 90_000;
/** "Silent Assist" trigger — see deep-improvement #4. */
export const SILENT_ASSIST_STALL_MS = 45_000;
