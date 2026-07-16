import { Component, type ErrorInfo, type ReactNode } from "react";

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    this.props.onError?.(error, errorInfo);
  }

  override render(): ReactNode {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <div role="alert" style={{ padding: "2rem", textAlign: "center" }}>
          <h2>Remote module failed to load</h2>
          <pre style={{ color: "#c00", fontSize: "0.875rem" }}>
            {this.state.error?.message ?? "Unknown error"}
          </pre>
          <button
            onClick={() => { this.setState({ hasError: false, error: null }); }}
          >
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
