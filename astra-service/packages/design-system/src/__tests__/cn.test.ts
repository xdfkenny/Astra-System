import { describe, it, expect } from "vitest";
import { cn } from "../utils/cn";

describe("cn", () => {
  it("merges conflicting Tailwind utility classes", () => {
    expect(cn("px-2 py-1", "px-4")).toBe("py-1 px-4");
  });

  it("handles conditional and falsy class values", () => {
    expect(cn("text-sm", false && "text-lg", "text-base")).toBe("text-base");
  });
});
