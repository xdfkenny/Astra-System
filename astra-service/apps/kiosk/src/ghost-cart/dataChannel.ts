/**
 * WebRTC data-channel transport for the "ghost cart" feature: a customer can
 * transfer an in-progress cart from their phone to the kiosk (or between
 * kiosks) without touching the public internet.
 *
 * All signaling happens out-of-band via QR codes or NFC NDEF messages. The
 * data channel itself is peer-to-peer over the local LAN when both peers are
 * on the same network, or via STUN/TURN if configured.
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

export function createGhostCartSession(): GhostCartSession {
  const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });
  const channel = pc.createDataChannel("ghost-cart", {
    ordered: true,
    maxRetransmits: 3,
  });

  const messageHandlers = new Set<(message: GhostCartMessage) => void>();
  const stateHandlers = new Set<(state: RTCPeerConnectionState) => void>();

  channel.addEventListener("message", (event) => {
    try {
      const parsed = JSON.parse(event.data as string) as GhostCartMessage;
      messageHandlers.forEach((h) => {
        h(parsed);
      });
    } catch {
      // Silently drop malformed ghost-cart messages.
    }
  });

  channel.addEventListener("open", () => {
    stateHandlers.forEach((h) => {
      h(pc.connectionState);
    });
  });

  pc.addEventListener("connectionstatechange", () => {
    stateHandlers.forEach((h) => {
      h(pc.connectionState);
    });
  });

  return {
    pc,
    send: (message) => {
      if (channel.readyState === "open") {
        channel.send(JSON.stringify(message));
      }
    },
    close: () => {
      channel.close();
      pc.close();
    },
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
    const check = () => {
      if (pc.iceGatheringState === "complete") {
        pc.removeEventListener("icegatheringstatechange", check);
        resolve();
      }
    };
    pc.addEventListener("icegatheringstatechange", check);
  });
}
