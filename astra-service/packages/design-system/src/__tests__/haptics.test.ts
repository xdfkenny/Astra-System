import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { haptic } from "../utils/haptics";

describe("haptics", () => {
  let originalVibrate: Navigator["vibrate"];

  beforeEach(() => {
    originalVibrate = navigator.vibrate;
  });

  afterEach(() => {
    navigator.vibrate = originalVibrate;
  });

  it("calls navigator.vibrate with the named pattern", () => {
    const vibrate = vi.fn(() => true);
    navigator.vibrate = vibrate as Navigator["vibrate"];

    const result = haptic("medium");

    expect(vibrate).toHaveBeenCalledWith([20]);
    expect(result).toBe(true);
  });

  it("returns false when vibrate is unavailable", () => {
    navigator.vibrate = undefined as unknown as Navigator["vibrate"];

    expect(haptic("light")).toBe(false);
  });
});
