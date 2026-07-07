import { ApolloClient, InMemoryCache, createHttpLink } from "@apollo/client";
import { setContext } from "@apollo/client/link/context";

const httpLink = createHttpLink({
  uri: (import.meta.env["VITE_ADMIN_GRAPHQL_URL"] as string | undefined) ?? "http://localhost:8080/v1/admin/graphql",
});

const authLink = setContext((_, { headers }: { headers?: Record<string, string> }) => {
  const token = localStorage.getItem("astra-admin-token");
  return {
    headers: {
      ...headers,
      authorization: token ? `Bearer ${token}` : "",
    },
  };
});

export const apolloClient = new ApolloClient({
  link: authLink.concat(httpLink),
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: { fetchPolicy: "cache-and-network" },
    query: { fetchPolicy: "network-only" },
  },
});
