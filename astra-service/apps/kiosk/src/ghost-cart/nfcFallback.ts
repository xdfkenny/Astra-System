import {
  validateQrSignalingEnvelope,
  type QrSignalingEnvelope,
} from "./qrSignaling";

/**
 * NFC NDEF fallback for ghost-cart signaling.
 *
 * When WebRTC cannot establish a direct connection (e.g. the phone has no
 * camera permission for QR scanning), the kiosk can write the offer into an
 * NDEF tag or receive an answer by reading one. This is guarded by feature
 * detection because NDEFReader is only available on a subset of Android
 * browsers and Chromium kiosk builds.
 */

export function isNfcAvailable(): boolean {
  return typeof window !== "undefined" && "NDEFReader" in window;
}

export async function writeNdefOffer(envelope: QrSignalingEnvelope): Promise<void> {
  if (!isNfcAvailable()) {
    throw new Error("NFC not available on this device");
  }
  const ndef = new NDEFReader();
  await ndef.write({
    records: [
      {
        recordType: "mime",
        mediaType: "application/vnd.astra.ghost-cart+json",
        data: new TextEncoder().encode(JSON.stringify(envelope)),
      },
    ],
  });
}

export async function readNdefAnswer(
  signal?: AbortSignal,
): Promise<QrSignalingEnvelope> {
  if (!isNfcAvailable()) {
    throw new Error("NFC not available on this device");
  }
  const ndef = new NDEFReader();
  return new Promise((resolve, reject) => {
    let settled = false;

    const onReading = (event: NDEFReadingEvent): void => {
      const record = event.message.records.find(
        (r) => r.mediaType === "application/vnd.astra.ghost-cart+json",
      );
      if (!record?.data) return;
      const bytes =
        record.data instanceof DataView
          ? new Uint8Array(record.data.buffer, record.data.byteOffset, record.data.byteLength)
          : new Uint8Array(record.data);
      const json = new TextDecoder().decode(bytes);
      try {
        const envelope = validateQrSignalingEnvelope(JSON.parse(json));
        settled = true;
        cleanup();
        resolve(envelope);
      } catch (error) {
        settled = true;
        cleanup();
        reject(error instanceof Error ? error : new Error(String(error)));
      }
    };

    const onError = (event: Event): void => {
      if (settled) return;
      settled = true;
      cleanup();
      const message = event instanceof ErrorEvent ? event.message : "NFC read error";
      reject(new Error(message));
    };

    const onAbort = (): void => {
      if (settled) return;
      settled = true;
      cleanup();
      reject(new DOMException("NFC read aborted", "AbortError"));
    };

    const cleanup = (): void => {
      ndef.removeEventListener("reading", onReading as EventListener);
      ndef.removeEventListener("error", onError);
      signal?.removeEventListener("abort", onAbort);
    };

    ndef.addEventListener("reading", onReading as EventListener);
    ndef.addEventListener("error", onError);
    signal?.addEventListener("abort", onAbort);

    ndef.scan().catch((err: unknown) => {
      if (settled) return;
      settled = true;
      cleanup();
      reject(err instanceof Error ? err : new Error(String(err)));
    });
  });
}

