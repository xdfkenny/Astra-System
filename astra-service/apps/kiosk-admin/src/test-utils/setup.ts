import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeAll, vi } from "vitest";
import type { FleetHealthSnapshot } from "../hooks/useFleetHealth";

beforeAll(() => {
  const emptyFleet: FleetHealthSnapshot = {
    nodes: [],
    paymentLanes: [],
    generatedAtMs: Date.now(),
  };
  vi.stubGlobal("fetch", vi.fn(() => Response.json(emptyFleet)));
});

afterEach(() => {
  cleanup();
});
