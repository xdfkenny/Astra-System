import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MockedProvider } from "@apollo/client/testing";
import { App } from "../App";
import { DASHBOARD_KPIS } from "../graphql/queries";

const testQueryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const mocks = [
  {
    request: { query: DASHBOARD_KPIS },
    result: {
      data: {
        dashboardKpis: {
          __typename: "DashboardKpis",
          totalRevenueCents: 0,
          orderCount: 0,
          activeKiosks: 0,
          alerts: 0,
          revenueTrend: 0,
          orderTrend: 0,
        },
      },
    },
  },
];

describe("App", () => {
  it("renders the dashboard route by default", () => {
    render(
      <MockedProvider mocks={mocks}>
        <QueryClientProvider client={testQueryClient}>
          <App />
        </QueryClientProvider>
      </MockedProvider>,
    );

    expect(screen.getByText("Astra Admin")).toBeInTheDocument();
  });
});
