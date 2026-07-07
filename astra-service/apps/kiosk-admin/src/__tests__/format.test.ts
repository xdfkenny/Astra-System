import { describe, expect, it } from "vitest";
import { formatCents, formatDate, formatDuration, formatPercent } from "../lib/format";

describe("formatCents", () => {
  it("formats cents as currency", () => {
    expect(formatCents(1234)).toBe("$12.34");
  });

  it("respects the provided currency", () => {
    expect(formatCents(999, "EUR")).toBe("€9.99");
  });
});

describe("formatDate", () => {
  it("formats an ISO string", () => {
    const result = formatDate("2024-06-15T14:30:00Z");
    expect(result).toContain("Jun");
    expect(result).toContain("2024");
  });
});

describe("formatPercent", () => {
  it("multiplies by 100 and appends %", () => {
    expect(formatPercent(0.1234)).toBe("12.3%");
  });
});

describe("formatDuration", () => {
  it("formats milliseconds", () => {
    expect(formatDuration(500)).toBe("500ms");
  });

  it("formats seconds", () => {
    expect(formatDuration(1500)).toBe("1.5s");
  });

  it("formats minutes", () => {
    expect(formatDuration(120_000)).toBe("2.0m");
  });
});
