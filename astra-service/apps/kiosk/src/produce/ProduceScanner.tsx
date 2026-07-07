import { useState } from "react";
import type { MenuItem } from "@astra/shared-types";
import { useProduceScanner } from "./useProduceScanner";

export interface ProduceScannerProps {
  readonly onLookupPlu: (plu: string) => MenuItem | null;
  readonly onSelectItem: (item: MenuItem) => void;
  readonly onClose: () => void;
}

/**
 * Produce scanner overlay. Shows the camera preview and a manual PLU fallback
 * entry. When recognition succeeds or a PLU is submitted, the matching catalog
 * item is selected and passed to the host.
 */
export function ProduceScanner({ onLookupPlu, onSelectItem, onClose }: ProduceScannerProps): React.JSX.Element {
  const { videoRef, state, startScanning, stopScanning, submitPlu } = useProduceScanner();
  const [plu, setPlu] = useState("");
  const [pluError, setPluError] = useState<string | null>(null);

  const handleStart = async (): Promise<void> => {
    await startScanning();
  };

  const handlePluSubmit = (): void => {
    const item = onLookupPlu(plu);
    if (item) {
      stopScanning();
      onSelectItem(item);
    } else {
      submitPlu(plu, []);
      setPluError(`PLU "${plu}" not found. Try ${["4011", "4013", "4062"].join(", ")}.`);
    }
  };

  return (
    <div className="absolute inset-0 z-modal flex flex-col bg-overlay p-6">
      <div className="flex flex-1 flex-col overflow-hidden rounded-xl bg-surface p-4">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-heading text-2xl font-bold text-ink">Scan Produce</h2>
          <button
            type="button"
            onClick={() => { stopScanning(); onClose(); }}
            aria-label="Close produce scanner"
            className="flex h-14 w-14 items-center justify-center rounded-full text-ink"
          >
            ×
          </button>
        </div>

        <div className="relative flex-1 overflow-hidden rounded-lg bg-black">
          {state.isScanning ? (
            <video
              ref={videoRef}
              autoPlay
              playsInline
              muted
              className="h-full w-full object-cover"
              aria-label="Produce camera preview"
            />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-4 text-center text-white">
              <p className="text-lg">Point the camera at fruits or vegetables.</p>
              <button
                type="button"
                onClick={() => { void handleStart(); }}
                className="rounded-lg bg-accent px-8 py-4 font-heading text-lg font-semibold text-ink"
              >
                Start Camera
              </button>
            </div>
          )}
        </div>

        {state.error ? (
          <p role="alert" className="mt-3 text-center text-error">{state.error}</p>
        ) : null}
        {state.lastMatch ? (
          <p className="mt-3 text-center text-success">Detected: {state.lastMatch.name} ({state.lastMatch.confidence}%)</p>
        ) : null}

        <div className="mt-4 flex flex-col gap-3">
          <label htmlFor="plu-input" className="text-sm font-medium text-ink-muted">
            Or enter PLU code manually
          </label>
          <div className="flex gap-2">
            <input
              id="plu-input"
              type="text"
              inputMode="numeric"
              value={plu}
              onChange={(e) => { setPlu(e.target.value); setPluError(null); }}
              placeholder="4011"
              className="hairline flex-1 rounded-lg bg-surface-sunken px-4 py-4 text-center text-2xl font-semibold tracking-widest text-ink"
              maxLength={4}
            />
            <button
              type="button"
              onClick={handlePluSubmit}
              disabled={plu.length === 0}
              className="rounded-lg bg-primary px-6 font-heading text-lg font-semibold text-white disabled:opacity-50"
            >
              Enter
            </button>
          </div>
          {pluError ? <p role="alert" className="text-error">{pluError}</p> : null}
        </div>
      </div>
    </div>
  );
}
