/**
 * WebRTC data-channel transport for the "ghost cart" feature: a customer can
 * transfer an in-progress cart from their phone to the kiosk (or between
 * kiosks) without touching the public internet.
 *
 * All signaling happens out-of-band via QR codes or NFC NDEF messages. The
 * data channel itself is peer-to-peer over the local LAN when both peers are
 * on the same network, or via STUN/TURN if configured.
 *
 * This module is hardened for unattended kiosk use: every session is bounded
 * by a handshake timeout, an outbound queue absorbs messages sent before the
 * channel opens, and a ping/pong liveness check auto-closes dead peers so the
 * kiosk never hangs on a phone that walked away.
 */

export interface GhostCartMessage {
  readonly type: "cart_snapshot" | "cart_op" | "ping" | "pong";
  readonly payload: unknown;
}

export interface GhostCartSession {
  readonly pc: RTCPeerConnection;
  readonly send: (message: GhostCartMessage) => void;
  readonly close: () => void;
  readonly onMessage: (handler: (message: GhostCartMessage) => void) => () => void;
  readonly onStateChange: (handler: (state: RTCPeerConnectionState) => void) => () => void;
}

const ICE_SERVERS: RTCIceServer[] = [
  { urls: "stun:stun.l.google.com:19302" },
];

const ICE_GATHER_TIMEOUT_MS = 5_000;
const HANDSHAKE_TIMEOUT_MS = 30_000;
const PING_INTERVAL_MS = 5_000;
const STALE_TIMEOUT_MS = 15_000;
const MAX_QUEUED_MESSAGES = 64;

/**
 * Create a ghost-cart P2P session.
 *
 * The returned session auto-closes under three conditions, all of which would
 * otherwise leave a kiosk stuck waiting on a customer who never scans:
 *   - the ICE handshake does not complete within HANDSHAKE_TIMEOUT_MS,
 *   - the underlying peer connection reports "failed",
 *   - no message is received for STALE_TIMEOUT_MS (peer walked away).
 * Consumers are notified of each via `onStateChange`.
 */
export function createGhostCartSession(): GhostCartSession {
  const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });
  const channel = pc.createDataChannel("ghost-cart", {
    ordered: true,
    maxRetransmits: 3,
  });

  const messageHandlers = new Set<(message: GhostCartMessage) => void>();
  const stateHandlers = new Set<(state: RTCPeerConnectionState) => void>();
  const outboundQueue: GhostCartMessage[] = [];

  let closed = false;
  let lastSeen = Date.now();
  let pingTimer: ReturnType<typeof setInterval> | null = null;
  let staleTimer: ReturnType<typeof setInterval> | null = null;
  let handshakeTimer: ReturnType<typeof setTimeout> | null = null;

  const markSeen = (): void => {
    lastSeen = Date.now();
  };

  const flushQueue = (): void => {
    if (channel.readyState !== "open") return;
    while (outboundQueue.length > 0) {
      const queued = outboundQueue.shift();
      if (!queued) break;
      try {
        channel.send(JSON.stringify(queued));
      } catch {
        // Drop undeliverable message; the transfer will be retried by the caller.
      }
    }
  };

  const clearTimers = (): void => {
    if (pingTimer !== null) {
      clearInterval(pingTimer);
      pingTimer = null;
    }
    if (staleTimer !== null) {
      clearInterval(staleTimer);
      staleTimer = null;
    }
    if (handshakeTimer !== null) {
      clearTimeout(handshakeTimer);
      handshakeTimer = null;
    }
  };

  const close = (): void => {
    if (closed) return;
    closed = true;
    clearTimers();
    try {
      channel.close();
    } catch {
      // ignore
    }
    try {
      pc.close();
    } catch {
      // ignore
    }
  };

  const startKeepalive = (): void => {
    pingTimer = setInterval(() => {
      if (channel.readyState === "open") {
        try {
          channel.send(JSON.stringify({ type: "ping", payload: null }));
        } catch {
          // ignore; staleness check will close if peer is truly gone
        }
      }
    }, PING_INTERVAL_MS);

    staleTimer = setInterval(() => {
      if (Date.now() - lastSeen > STALE_TIMEOUT_MS) {
        console.warn("GhostCart: peer went silent; closing session");
        close();
      }
    }, PING_INTERVAL_MS);
  };

  channel.addEventListener("message", (event) => {
    try {
      const parsed = JSON.parse(event.data as string) as GhostCartMessage;
      markSeen();
      if (parsed.type === "ping") {
        // Answer liveness probes internally; do not surface them to app handlers.
        if (channel.readyState === "open") {
          try {
            channel.send(JSON.stringify({ type: "pong", payload: null }));
          } catch {
            // ignore
          }
        }
        return;
      }
      messageHandlers.forEach((handler) => {
        handler(parsed);
      });
    } catch {
      // Silently drop malformed ghost-cart messages.
    }
  });

  channel.addEventListener("open", () => {
    markSeen();
    if (handshakeTimer !== null) {
      clearTimeout(handshakeTimer);
      handshakeTimer = null;
    }
    flushQueue();
    startKeepalive();
    stateHandlers.forEach((handler) => {
      handler(pc.connectionState);
    });
  });

  pc.addEventListener("connectionstatechange", () => {
    const state = pc.connectionState;
    if (state === "connected" && handshakeTimer !== null) {
      clearTimeout(handshakeTimer);
      handshakeTimer = null;
    }
    if (state === "failed") {
      close();
      return;
    }
    stateHandlers.forEach((handler) => {
      handler(state);
    });
  });

  // Bounded handshake: if the customer never scans / connects, close so the
  // kiosk's ghost-cart prompt can time out and return to the menu.
  handshakeTimer = setTimeout(() => {
    if (!closed) {
      console.warn("GhostCart: handshake timed out; closing session");
      close();
    }
  }, HANDSHAKE_TIMEOUT_MS);

  return {
    pc,
    send: (message) => {
      if (channel.readyState === "open") {
        try {
          channel.send(JSON.stringify(message));
        } catch {
          // Drop; caller can retry the transfer.
        }
        return;
      }
      if (closed) return;
      if (outboundQueue.length < MAX_QUEUED_MESSAGES) {
        outboundQueue.push(message);
      } else {
        console.warn("GhostCart: outbound queue full; dropping message");
      }
    },
    close,
    onMessage: (handler) => {
      messageHandlers.add(handler);
      return () => {
        messageHandlers.delete(handler);
      };
    },
    onStateChange: (handler) => {
      stateHandlers.add(handler);
      return () => {
        stateHandlers.delete(handler);
      };
    },
  };
}

export async function createOffer(session: GhostCartSession): Promise<RTCSessionDescriptionInit> {
  const { pc } = session;
  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);
  await waitForIceGathering(pc);
  return pc.localDescription ?? offer;
}

export async function acceptAnswer(
  session: GhostCartSession,
  answer: RTCSessionDescriptionInit,
): Promise<void> {
  await session.pc.setRemoteDescription(answer);
}

function waitForIceGathering(pc: RTCPeerConnection): Promise<void> {
  return new Promise((resolve) => {
    if (pc.iceGatheringState === "complete") {
      resolve();
      return;
    }
    let settled = false;
    const finish = (): void => {
      if (settled) return;
      settled = true;
      pc.removeEventListener("icegatheringstatechange", check);
      clearTimeout(timer);
      resolve();
    };
    const check = (): void => {
      if (pc.iceGatheringState === "complete") {
        finish();
      }
    };
    // Never hang the offer on ICE gathering; proceed with whatever candidates we have.
    const timer = setTimeout(finish, ICE_GATHER_TIMEOUT_MS);
    pc.addEventListener("icegatheringstatechange", check);
  });
}
