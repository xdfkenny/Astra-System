import { describe, expect, it, vi } from "vitest";
import { authenticateEmployee, isWebAuthnAvailable } from "./employeeAuth";

describe("employeeAuth", () => {
  it("reports availability based on PublicKeyCredential", () => {
    expect(isWebAuthnAvailable()).toBe(typeof window.PublicKeyCredential !== "undefined");
  });

  it("resolves with base64url fields when credentials.get succeeds", async () => {
    vi.stubGlobal("PublicKeyCredential", function PublicKeyCredentialStub() {
      /* stub for feature-detection tests */
    });

    const mockResponse = {
      authenticatorData: new ArrayBuffer(8),
      clientDataJSON: new ArrayBuffer(8),
      signature: new ArrayBuffer(8),
      userHandle: new ArrayBuffer(8),
    } as unknown as AuthenticatorAssertionResponse;

    vi.stubGlobal("navigator", {
      credentials: {
        get: vi.fn().mockResolvedValue({ rawId: new ArrayBuffer(8), response: mockResponse }),
      },
    });

    const result = await authenticateEmployee({ challenge: new Uint8Array(8) });

    expect(result.credentialId).toBeTypeOf("string");
    expect(result.signature).toBeTypeOf("string");

    vi.unstubAllGlobals();
  });
});
