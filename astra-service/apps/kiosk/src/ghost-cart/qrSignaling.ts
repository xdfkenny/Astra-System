import QRCode from "qrcode";

/**
 * QR-code signaling for ghost-cart WebRTC offers.
 *
 * The offer JSON is encoded as a base64url payload in a `astra://ghost-cart#`
 * URL. The receiving peer scans the QR code, extracts the payload, and posts
 * the answer back via its own QR code or NFC tag.
 */

export interface QrSignalingEnvelope {
  readonly type: "offer" | "answer";
  readonly sdp: string;
  readonly kioskId: string;
}

export class GhostCartSignalingError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "GhostCartSignalingError";
  }
}

export async function generateQrDataUrl(
  envelope: QrSignalingEnvelope,
  options?: QRCode.QRCodeToDataURLOptions,
): Promise<string> {
  let payload: string;
  try {
    payload = btoa(JSON.stringify(envelope));
  } catch {
    throw new GhostCartSignalingError("Failed to encode ghost-cart offer");
  }
  const url = `astra://ghost-cart#${payload}`;
  return QRCode.toDataURL(url, {
    width: 320,
    margin: 2,
    errorCorrectionLevel: "M",
    ...options,
  });
}

export function parseQrPayload(raw: string): QrSignalingEnvelope {
  const hashIndex = raw.indexOf("#");
  const payload = hashIndex >= 0 ? raw.slice(hashIndex + 1) : raw;
  let decoded: string;
  try {
    decoded = atob(payload);
  } catch {
    throw new GhostCartSignalingError("Invalid ghost-cart QR encoding");
  }
  return parseEnvelope(decoded);
}

export function validateQrSignalingEnvelope(value: unknown): QrSignalingEnvelope {
  if (!isEnvelope(value)) {
    throw new GhostCartSignalingError("Invalid ghost-cart envelope");
  }
  return value;
}

function parseEnvelope(json: string): QrSignalingEnvelope {
  let data: unknown;
  try {
    data = JSON.parse(json);
  } catch {
    throw new GhostCartSignalingError("Malformed ghost-cart payload");
  }
  return validateQrSignalingEnvelope(data);
}

function isEnvelope(value: unknown): value is QrSignalingEnvelope {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Record<string, unknown>;
  return (
    (candidate["type"] === "offer" || candidate["type"] === "answer") &&
    typeof candidate["sdp"] === "string" &&
    typeof candidate["kioskId"] === "string"
  );
}
