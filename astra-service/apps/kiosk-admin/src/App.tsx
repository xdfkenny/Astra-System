import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApolloProvider } from "@apollo/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { apolloClient } from "./lib/apollo";
import { AuthProvider } from "./components/AuthProvider";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { RouteGuard } from "./components/RouteGuard";
import { Dashboard } from "./routes/Dashboard";
import { Locations } from "./routes/Locations";
import { Lanes } from "./routes/Lanes";
import { Kiosks } from "./routes/Kiosks";
import { Menu } from "./routes/Menu";
import { Inventory } from "./routes/Inventory";
import { Orders } from "./routes/Orders";
import { PaymentsRefunds } from "./routes/PaymentsRefunds";
import { EmployeesRoles } from "./routes/EmployeesRoles";
import { AuditLogs } from "./routes/AuditLogs";
import "./styles/global.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 10_000, refetchOnWindowFocus: false },
  },
});

function AdminRoutes(): React.JSX.Element {
  return (
    <Routes>
      <Route path="/" element={<Dashboard />} />
      <Route
        path="/locations"
        element={
          <RouteGuard resource="locations">
            <Locations />
          </RouteGuard>
        }
      />
      <Route
        path="/lanes"
        element={
          <RouteGuard resource="lanes">
            <Lanes />
          </RouteGuard>
        }
      />
      <Route
        path="/kiosks"
        element={
          <RouteGuard resource="kiosks">
            <Kiosks />
          </RouteGuard>
        }
      />
      <Route
        path="/menu"
        element={
          <RouteGuard resource="menu">
            <Menu />
          </RouteGuard>
        }
      />
      <Route
        path="/inventory"
        element={
          <RouteGuard resource="inventory">
            <Inventory />
          </RouteGuard>
        }
      />
      <Route
        path="/orders"
        element={
          <RouteGuard resource="orders">
            <Orders />
          </RouteGuard>
        }
      />
      <Route
        path="/payments"
        element={
          <RouteGuard resource="payments">
            <PaymentsRefunds />
          </RouteGuard>
        }
      />
      <Route
        path="/employees"
        element={
          <RouteGuard resource="employees">
            <EmployeesRoles />
          </RouteGuard>
        }
      />
      <Route
        path="/audit"
        element={
          <RouteGuard resource="audit">
            <AuditLogs />
          </RouteGuard>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export function App(): React.JSX.Element {
  return (
    <ErrorBoundary>
      <ApolloProvider client={apolloClient}>
        <QueryClientProvider client={queryClient}>
          <AuthProvider initialRole="admin">
            <BrowserRouter
              future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
            >
              <AdminRoutes />
            </BrowserRouter>
          </AuthProvider>
        </QueryClientProvider>
      </ApolloProvider>
    </ErrorBoundary>
  );
}

export default App;
