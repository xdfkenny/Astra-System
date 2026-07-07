import { Component, type ErrorInfo, type ReactNode } from "react";

interface ErrorBoundaryProps {
  readonly children: ReactNode;
}

interface ErrorBoundaryState {
  readonly error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Admin app error boundary caught an error:", error, info);
  }

  override render() {
    if (this.state.error) {
      return (
        <div className="flex h-full flex-col items-center justify-center gap-4 p-6 text-center">
          <h2 className="font-heading text-2xl font-bold text-error">Something went wrong</h2>
          <p className="text-ink-muted">{this.state.error.message}</p>
          <button
            type="button"
            onClick={() => { this.setState({ error: null }); }}
            className="rounded-md bg-primary px-4 py-2 text-white hover:bg-primary-hover"
          >
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
