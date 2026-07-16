import { lazy, Suspense, type ComponentType } from "react";
import { ErrorBoundary } from "./ErrorBoundary";
import type { RemoteModuleManager } from "./remote-modules";

type ImportFactory<T> = () => Promise<{ default: ComponentType<T> }>;

interface RemoteModuleOptions {
  manager: RemoteModuleManager;
  remoteName: string;
  fallback?: React.ReactNode;
}

export function withRemoteModule<T extends object>(
  importFactory: ImportFactory<T>,
  options: RemoteModuleOptions,
): ComponentType<T> {
  const LazyComponent = lazy(importFactory);

  function Wrapped(props: T) {
    return (
      <ErrorBoundary
        onError={(_error) => {
          options.manager.rollbackAll();
        }}
        fallback={options.fallback ?? (
          <div role="alert" style={{ padding: "2rem", textAlign: "center" }}>
            <h2>{options.remoteName} unavailable</h2>
            <p>This module failed to load. Rolling back to previous version.</p>
          </div>
        )}
      >
        <Suspense
          fallback={
            <div
              style={{
                padding: "2rem",
                textAlign: "center",
                color: "#666",
              }}
              aria-busy="true"
            >
              Loading {options.remoteName}...
            </div>
          }
        >
          <LazyComponent {...props} />
        </Suspense>
      </ErrorBoundary>
    );
  }

  return Wrapped;
}
