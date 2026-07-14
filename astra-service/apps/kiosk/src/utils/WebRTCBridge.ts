import type { CartLineItem } from "@astra/shared-types";

const ENV = import.meta.env as Record<string, string | undefined>;
const ASTRA_NAMESPACE = "astra-kiosk-p2p";

export interface GhostCartPayload {
  readonly storeId: string;
  readonly kioskId: string;
  readonly cartId: string;
  readonly lines: readonly CartLineItem[];
  readonly version: number;
  readonly transferredAt: string;
}

export type GhostCartCallback = (payload: GhostCartPayload) => void;

type SignalingMessage =
  | { readonly type: "offer"; readonly offer: RTCSessionDescriptionInit }
  | { readonly type: "ice-candidate"; readonly candidate: RTCIceCandidateInit }
  | { readonly type: "ghost-cart-transfer"; readonly payload: GhostCartPayload };

export class WebRTCBridge {
  private peerConnection: RTCPeerConnection | null = null;
  private dataChannel: RTCDataChannel | null = null;
  private callbacks: Set<GhostCartCallback> = new Set<GhostCartCallback>();
  private signalingUrl: string | null;
  private _isConnected = false;
  private kioskId: string;

  constructor(kioskId?: string) {
    this.kioskId = kioskId ?? ENV["VITE_KIOSK_ID"] ?? "kiosk-local";
    this.signalingUrl = ENV["VITE_SIGNALING_SERVER_URL"] ?? null;
  }

  get isConnected(): boolean {
    return this._isConnected;
  }

  onGhostCart(cb: GhostCartCallback): () => void {
    this.callbacks.add(cb);
    return () => {
      this.callbacks.delete(cb);
    };
  }

  async connect(): Promise<void> {
    if (this._isConnected) return;

    const iceServers: RTCIceServer[] = [
      { urls: "stun:stun.l.google.com:19302" },
    ];
    const turnUrl = ENV["VITE_TURN_SERVER_URL"];
    const turnUsername = ENV["VITE_TURN_USERNAME"];
    const turnCredential = ENV["VITE_TURN_CREDENTIAL"];
    if (turnUrl && turnCredential) {
      const server: RTCIceServer = {
        urls: turnUrl,
        credential: turnCredential,
      };
      if (turnUsername) {
        server.username = turnUsername;
      }
      iceServers.push(server);
    }

    this.peerConnection = new RTCPeerConnection({ iceServers });

    this.peerConnection.oniceconnectionstatechange = () => {
      this._isConnected =
        this.peerConnection?.iceConnectionState === "connected";
    };

    if (this.signalingUrl) {
      await this.connectViaSignaling();
    } else {
      this.createOffer();
    }
  }

  private async connectViaSignaling(): Promise<void> {
    if (!this.peerConnection || !this.signalingUrl) return;

    const ws = new WebSocket(`${this.signalingUrl}?kioskId=${this.kioskId}`);
    await new Promise<void>((resolve, reject) => {
      ws.onopen = () => {
        resolve();
      };
      ws.onerror = () => {
        reject(new Error("Failed to connect to signaling server"));
      };
    });

    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "ice-candidate", candidate: event.candidate }));
      }
    };

    ws.onmessage = async (event) => {
      const msg = JSON.parse(event.data as string) as SignalingMessage;
      if (msg.type === "offer") {
        await this.peerConnection?.setRemoteDescription(new RTCSessionDescription(msg.offer));
        const answer = await this.peerConnection?.createAnswer();
        if (answer) {
          await this.peerConnection?.setLocalDescription(answer);
          ws.send(JSON.stringify({ type: "answer", answer }));
        }
      } else if (msg.type === "ice-candidate") {
        await this.peerConnection?.addIceCandidate(new RTCIceCandidate(msg.candidate));
      } else {
        this.handleGhostCartPayload(msg.payload);
      }
    };
  }

  private createOffer(): void {
    if (!this.peerConnection) return;
    this.dataChannel = this.peerConnection.createDataChannel(ASTRA_NAMESPACE);
    this.setupDataChannel();
  }

  private setupDataChannel(): void {
    if (!this.dataChannel) return;
    this.dataChannel.onopen = () => {
      this._isConnected = true;
    };
    this.dataChannel.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data as string) as GhostCartPayload;
        this.handleGhostCartPayload(payload);
      } catch {
        // Ignore malformed messages
      }
    };
  }

  private handleGhostCartPayload(payload: GhostCartPayload): void {
    for (const cb of this.callbacks) {
      cb(payload);
    }
  }

  async transferGhostCart(payload: GhostCartPayload): Promise<void> {
    if (this.dataChannel?.readyState === "open") {
      this.dataChannel.send(JSON.stringify(payload));
    }

    if (this.signalingUrl) {
      try {
        const response = await fetch(`${this.signalingUrl}/transfer`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ from: this.kioskId, payload }),
        });
        if (!response.ok) {
          throw new Error(`Transfer failed: ${response.status}`);
        }
      } catch {
        // Silent fallback
      }
    }
  }

  disconnect(): void {
    this.dataChannel?.close();
    this.peerConnection?.close();
    this.peerConnection = null;
    this.dataChannel = null;
    this._isConnected = false;
  }
}

let bridgeInstance: WebRTCBridge | null = null;

export function getWebRTCBridge(): WebRTCBridge {
  bridgeInstance ??= new WebRTCBridge();
  return bridgeInstance;
}

