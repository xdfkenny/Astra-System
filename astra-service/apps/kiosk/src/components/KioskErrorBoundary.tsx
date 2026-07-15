import { Component, type ErrorInfo, type ReactNode } from "react";
import { defaultLogger } from "../utils/logger";

const log = defaultLogger.child("KioskErrorBoundary");

interface KioskErrorBoundaryProps {
  readonly children: ReactNode;
  readonly fallback?: ReactNode;
}

interface KioskErrorBoundaryState {
  readonly hasError: boolean;
  readonly error: Error | null;
}

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

  override componentDidCatch(error: Error, _errorInfo: ErrorInfo): void {
    log.error("KioskErrorBoundary caught error", error, {
      componentStack: _errorInfo.componentStack,
    });
  }

  override render(): ReactNode {
    if (this.state.hasError) {
      return (
        this.props.fallback ?? (
          <div
            className="flex flex-1 flex-col items-center justify-center bg-linen p-6 text-center safe-bottom safe-top"
            role="alert"
            aria-live="assertive"
          >
            <h1 className="font-heading text-[36px] font-semibold text-charcoal">
              Something went wrong
            </h1>
            <p className="mt-3 font-sans text-[18px] text-stone max-w-sm">
              Please ask a staff member for help. We apologise for the inconvenience.
            </p>
            {import.meta.env.DEV && this.state.error && (
              <pre
                className="mt-4 max-w-full overflow-auto rounded-[12px] bg-white/50 p-3 font-mono text-[14px] text-charcoal border border-taupe"
                aria-label="Error details"
              >
                {this.state.error.message}
                {this.state.error.stack && (
                  <>
                    {"\n\n"}
                    {this.state.error.stack}
                  </>
                )}
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
              className="mt-6 h-14 min-w-[56px] rounded-full bg-amber px-8 font-sans text-[18px] font-medium text-white shadow-[0_4px_16px_rgba(184,126,107,0.3)] active:scale-[0.98] active:translate-y-[1px] transition-all duration-100"
              aria-label="Restart kiosk"
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

