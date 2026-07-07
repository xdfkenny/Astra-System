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

export async function generateQrDataUrl(
  envelope: QrSignalingEnvelope,
  options?: QRCode.QRCodeToDataURLOptions,
): Promise<string> {
  const payload = btoa(JSON.stringify(envelope));
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
  const decoded = atob(payload);
  return JSON.parse(decoded) as QrSignalingEnvelope;
}
