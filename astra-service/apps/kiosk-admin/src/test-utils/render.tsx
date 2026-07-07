import { render as tlRender } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApolloProvider } from "@apollo/client";
import { AuthProvider } from "../components/AuthProvider";
import { apolloClient } from "../lib/apollo";

const testQueryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false },
    mutations: { retry: false },
  },
});

export function render(ui: ReactElement, { route = "/" }: { route?: string } = {}) {
  return tlRender(
    <MemoryRouter initialEntries={[route]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <ApolloProvider client={apolloClient}>
        <QueryClientProvider client={testQueryClient}>
          <AuthProvider initialRole="admin">{ui as ReactNode}</AuthProvider>
        </QueryClientProvider>
      </ApolloProvider>
    </MemoryRouter>,
  );
}
