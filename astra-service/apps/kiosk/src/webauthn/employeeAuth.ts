/**
 * WebAuthn integration for employee override / supervisor authorization.
 *
 * Kiosk employees authenticate with a FIDO2 security key or platform
 * authenticator before performing high-privilege actions (voids, refunds,
 * manual lane unlock). No password or shared PIN is ever typed into the
 * touchscreen; the auth factor is bound to the employee's authenticator.
 */

export interface EmployeeAuthenticationResult {
  readonly credentialId: string;
  readonly authenticatorData: string;
  readonly clientDataJSON: string;
  readonly signature: string;
  readonly userHandle: string | null;
}

/**
 * Starts an employee authentication ceremony. In production this receives the
 * challenge from the local astra-authd sidecar (which validates the employee
 * against the identity store); here we accept the raw PublicKeyCredentialRequestOptions.
 */
export async function authenticateEmployee(
  options: PublicKeyCredentialRequestOptions,
): Promise<EmployeeAuthenticationResult> {
  if (!("PublicKeyCredential" in window)) {
    throw new Error("WebAuthn is not supported on this device");
  }

  const credential = (await navigator.credentials.get({ publicKey: options })) as PublicKeyCredential | null;
  if (credential === null) {
    throw new Error("Employee authentication was cancelled or failed");
  }

  const response = credential.response as AuthenticatorAssertionResponse;
  return {
    credentialId: arrayBufferToBase64Url(credential.rawId),
    authenticatorData: arrayBufferToBase64Url(response.authenticatorData),
    clientDataJSON: arrayBufferToBase64Url(response.clientDataJSON),
    signature: arrayBufferToBase64Url(response.signature),
    userHandle: response.userHandle ? arrayBufferToBase64Url(response.userHandle) : null,
  };
}

/**
 * Returns whether the current device can perform WebAuthn. Useful for falling
 * back to a supervisor card tap when the browser has no authenticator.
 */
export function isWebAuthnAvailable(): boolean {
  return typeof window !== "undefined" && Boolean(window.PublicKeyCredential);
}

function arrayBufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

