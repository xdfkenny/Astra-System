import { useCallback, useState } from "react";

/**
 * Deep-improvement #6: WebAuthn/Passkeys for employee authentication
 * (supervisor override — e.g. age-restricted item confirmation, price
 * override, void). Replaces password-based employee logins entirely.
 * Biometric verification is tied to the Verifone PIN pad's fingerprint
 * reader where equipped (exposed to the browser as a platform authenticator
 * via the kiosk OS's WebAuthn provider) or the kiosk's own platform sensor.
 */
export interface EmployeeAuthResult {
  readonly employeeIdHash: string; // irreversible hash, never a raw ID — see data model mandate
  readonly credentialId: string;
}

export function useWebAuthnEmployeeAuth(): {
  authenticate: (challengeFromServer: string) => Promise<EmployeeAuthResult>;
  isAuthenticating: boolean;
  error: string | null;
} {
  const [isAuthenticating, setIsAuthenticating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const authenticate = useCallback(async (challengeFromServer: string): Promise<EmployeeAuthResult> => {
    setIsAuthenticating(true);
    setError(null);
    try {
      if (!("credentials" in navigator)) {
        throw new Error("WebAuthn not supported on this kiosk browser build");
      }

      const publicKey: PublicKeyCredentialRequestOptions = {
        challenge: base64UrlToBuffer(challengeFromServer),
        timeout: 20_000,
        userVerification: "required", // forces biometric/PIN, not just "presence"
        rpId: window.location.hostname,
      };

      const credential = await navigator.credentials.get({ publicKey });
      if (!credential || !("rawId" in credential)) {
        throw new Error("Authentication cancelled or failed");
      }
      const pk = credential as PublicKeyCredential;

      return {
        // The employeeId->hash mapping is resolved server-side from the credential ID;
        // the browser never learns or stores the raw employee identifier.
        employeeIdHash: await sha256Hex(pk.rawId),
        credentialId: bufferToBase64Url(pk.rawId),
      };
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown WebAuthn error";
      setError(message);
      throw err;
    } finally {
      setIsAuthenticating(false);
    }
  }, []);

  return { authenticate, isAuthenticating, error };
}

function base64UrlToBuffer(base64Url: string): ArrayBuffer {
  const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=");
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}

function bufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

async function sha256Hex(buffer: ArrayBuffer): Promise<string> {
  const hash = await crypto.subtle.digest("SHA-256", buffer);
  return Array.from(new Uint8Array(hash))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
