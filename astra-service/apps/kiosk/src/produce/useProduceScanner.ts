import { useCallback, useEffect, useRef, useState } from "react";
import type { MenuItem } from "@astra/shared-types";
import { createOnnxProduceRecognizer, lookupByPlu, matchToMenuItem } from "@astra/cart-engine";
import type { ProduceMatch, ProduceRecognizer } from "@astra/cart-engine";

export interface ProduceScannerState {
  readonly isScanning: boolean;
  readonly error: string | null;
  readonly lastMatch: ProduceMatch | null;
}

export interface UseProduceScannerResult {
  readonly videoRef: React.RefObject<HTMLVideoElement | null>;
  readonly state: ProduceScannerState;
  readonly startScanning: () => Promise<void>;
  readonly stopScanning: () => void;
  readonly submitPlu: (plu: string, catalog: readonly MenuItem[]) => MenuItem | null;
}

/**
 * Manages the kiosk camera stream for produce recognition. Captures frames from
 * the video element and passes them to the ONNX recognizer. When confidence is
 * too low or the camera is unavailable, the manual PLU fallback is used.
 */
export function useProduceScanner(): UseProduceScannerResult {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const recognizerRef = useRef<ProduceRecognizer | null>(null);
  const rafRef = useRef<number | null>(null);
  const [state, setState] = useState<ProduceScannerState>({
    isScanning: false,
    error: null,
    lastMatch: null,
  });

  useEffect(() => {
    recognizerRef.current = createOnnxProduceRecognizer();
    return () => {
      stopScanning();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const stopScanning = useCallback((): void => {
    if (rafRef.current) {
      cancelAnimationFrame(rafRef.current);
      rafRef.current = null;
    }
    streamRef.current?.getTracks().forEach((track) => {
      track.stop();
    });
    streamRef.current = null;
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
    setState((prev) => ({ ...prev, isScanning: false }));
  }, []);

  const captureAndRecognize = useCallback(async (): Promise<void> => {
    const video = videoRef.current;
    const recognizer = recognizerRef.current;
    if (!video || !recognizer || video.readyState < video.HAVE_CURRENT_DATA) {
      rafRef.current = requestAnimationFrame(() => { void captureAndRecognize(); });
      return;
    }

    try {
      const bitmap = await createImageBitmap(video);
      const result = await recognizer.recognize(bitmap);
      if (result.bestMatch) {
        setState((prev) => ({ ...prev, lastMatch: result.bestMatch }));
      }
    } catch (err) {
      setState((prev) => ({
        ...prev,
        error: err instanceof Error ? err.message : "Produce recognition failed.",
      }));
    } finally {
      rafRef.current = requestAnimationFrame(() => { void captureAndRecognize(); });
    }
  }, []);

  const startScanning = useCallback(async (): Promise<void> => {
    setState({ isScanning: true, error: null, lastMatch: null });
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "environment", width: { ideal: 1280 }, height: { ideal: 720 } },
        audio: false,
      });
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
      }
      rafRef.current = requestAnimationFrame(() => { void captureAndRecognize(); });
    } catch (err) {
      setState((prev) => ({
        ...prev,
        isScanning: false,
        error: err instanceof Error ? err.message : "Camera unavailable.",
      }));
    }
  }, [captureAndRecognize]);

  const submitPlu = useCallback((plu: string, catalog: readonly MenuItem[]): MenuItem | null => {
    const match = lookupByPlu(plu);
    if (!match) return null;
    setState((prev) => ({ ...prev, lastMatch: match }));
    return matchToMenuItem(match, catalog);
  }, []);

  return {
    videoRef,
    state,
    startScanning,
    stopScanning,
    submitPlu,
  };
}
