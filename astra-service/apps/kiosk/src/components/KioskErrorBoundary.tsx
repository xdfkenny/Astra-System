import { Component, type ErrorInfo, type ReactNode } from "react";

interface KioskErrorBoundaryProps {
  readonly children: ReactNode;
  readonly fallback?: ReactNode;
}

interface KioskErrorBoundaryState {
  readonly hasError: boolean;
  readonly error: Error | null;
}

/**
 * Catches render errors anywhere in the kiosk workflow and prevents a white
 * screen of death. In production it shows a calm recovery UI; in development it
 * also prints the error message so engineers can diagnose at the terminal.
 */
export class KioskErrorBoundary extends Component<
  KioskErrorBoundaryProps,
  KioskErrorBoundaryState
> {
  constructor(props: KioskErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): KioskErrorBoundaryState {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error("KioskErrorBoundary caught error:", error, errorInfo);
  }

  override render(): ReactNode {
    if (this.state.hasError) {
      return (
        this.props.fallback ?? (
          <div className="flex flex-1 flex-col items-center justify-center bg-linen p-6 text-center safe-bottom safe-top">
            <h1 className="font-heading text-title font-semibold text-charcoal">
              Something went wrong
            </h1>
            <p className="mt-3 font-sans text-body text-stone">
              Please ask a staff member for help.
            </p>
            {import.meta.env.DEV && this.state.error && (
              <pre className="mt-4 max-w-full overflow-auto rounded bg-white/50 p-3 font-mono text-micro text-charcoal">
                {this.state.error.message}
              </pre>
            )}
            <button
              type="button"
              onPointerDown={() => {
                window.location.reload();
              }}
              onClick={() => {
                window.location.reload();
              }}
              className="mt-6 h-14 min-w-[56px] rounded-full bg-amber px-8 font-sans text-[18px] font-medium text-white shadow-[0_4px_16px_rgba(184,126,107,0.3)] tap-feedback"
            >
              Restart kiosk
            </button>
          </div>
        )
      );
    }

    return this.props.children;
  }
}
