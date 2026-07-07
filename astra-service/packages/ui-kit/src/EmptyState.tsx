export interface EmptyStateProps {
  readonly title: string;
  readonly description: string;
  readonly actionLabel?: string;
  readonly onAction?: () => void;
}

/** Reusable empty-state block — never a bare "No items" text per the UX spec. */
export function EmptyState({
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateProps): React.JSX.Element {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
      <svg
        viewBox="0 0 64 64"
        className="h-16 w-16 text-secondary"
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        aria-hidden="true"
      >
        <rect x="10" y="20" width="44" height="34" rx="4" />
        <path d="M10 28h44" />
        <circle cx="22" cy="40" r="3" fill="currentColor" stroke="none" />
        <circle cx="42" cy="40" r="3" fill="currentColor" stroke="none" />
      </svg>
      <h3 className="font-heading text-xl font-semibold text-ink">{title}</h3>
      <p className="max-w-xs text-ink-muted">{description}</p>
      {actionLabel && onAction ? (
        <button
          type="button"
          onClick={onAction}
          className="mt-2 h-12 rounded-md bg-primary px-6 font-medium text-white"
        >
          {actionLabel}
        </button>
      ) : null}
    </div>
  );
}
