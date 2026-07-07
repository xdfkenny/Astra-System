import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { render } from "../test-utils/render";
import { Dashboard } from "../routes/Dashboard";
import type * as ReactQuery from "@tanstack/react-query";
import type * as ApolloClient from "@apollo/client";

const mockUseQuery = vi.fn<() => unknown>();

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof ReactQuery>("@tanstack/react-query");
  return {
    ...actual,
    useQuery: () => mockUseQuery(),
  };
});

vi.mock("@apollo/client", async () => {
  const actual = await vi.importActual<typeof ApolloClient>("@apollo/client");
  return {
    ...actual,
    useQuery: () => ({
      data: {
        dashboardKpis: {
          totalRevenueCents: 123456,
          orderCount: 42,
          activeKiosks: 5,
          alerts: 0,
          revenueTrend: 5.2,
          orderTrend: 3,
        },
      },
      loading: false,
      error: undefined,
    }),
  };
});

describe("Dashboard", () => {
  it("renders KPI cards and mesh sections", () => {
    mockUseQuery.mockReturnValue({
      data: {
        nodes: [
          {
            kioskId: "K1",
            storeId: "S1",
            health: "healthy" as const,
            isLeader: true,
            syncLagMs: 10,
            paymentSuccessRate: 0.99,
            meshPeers: ["K2"],
          },
        ],
        paymentLanes: [
          { laneId: "L1", circuitState: "closed" as const, consecutiveFailures: 0, lastFailureReason: null },
        ],
        generatedAtMs: Date.now(),
      },
      isLoading: false,
      isError: false,
    });

    render(<Dashboard />);

    expect(screen.getByRole("heading", { name: "Revenue" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Orders" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Mesh Topology" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Payment Lane Circuit Breakers" })).toBeInTheDocument();
  });
});
