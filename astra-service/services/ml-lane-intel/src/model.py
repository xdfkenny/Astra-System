"""ONNX Runtime inference wrapper for YOLOv8n person detection and queue depth."""

from __future__ import annotations

import io
import logging
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import BinaryIO

import numpy as np
import onnxruntime as ort
from PIL import Image

logger = logging.getLogger(__name__)

INPUT_SIZE = 640
PERSON_CLASS_ID = 0
DEFAULT_CONFIDENCE = 0.25
DEFAULT_IOU = 0.45
SECONDS_PER_PERSON = 45


@dataclass(frozen=True, slots=True)
class BoundingBox:
    """An axis-aligned bounding box in normalized [0, 1] coordinates."""

    x1: float
    y1: float
    x2: float
    y2: float

    @property
    def width(self) -> float:
        return max(0.0, self.x2 - self.x1)

    @property
    def height(self) -> float:
        return max(0.0, self.y2 - self.y1)

    @property
    def area(self) -> float:
        return self.width * self.height

    @property
    def center(self) -> tuple[float, float]:
        return (self.x1 + self.width * 0.5, self.y1 + self.height * 0.5)


@dataclass(frozen=True, slots=True)
class Detection:
    """A single person detection."""

    label: str
    confidence: float
    box: BoundingBox


@dataclass(frozen=True, slots=True)
class QueueDepthResult:
    """Queue-depth estimate for one frame."""

    queue_depth: int
    total_people: int
    people_in_roi: int
    people_outside_roi: int
    estimated_wait_seconds: int
    detections: list[Detection]


def decode_image(image_bytes: bytes | BinaryIO) -> np.ndarray:
    """Decode raw image bytes into a RGB numpy array (H, W, 3)."""
    if isinstance(image_bytes, bytes):
        image_bytes = io.BytesIO(image_bytes)
    image = Image.open(image_bytes)
    if image.mode != "RGB":
        image = image.convert("RGB")
    return np.array(image, dtype=np.uint8)


def _letterbox(
    image: np.ndarray,
    target_size: int = INPUT_SIZE,
    pad_color: tuple[int, int, int] = (114, 114, 114),
) -> tuple[np.ndarray, float, tuple[float, float]]:
    """Resize and pad an image to a square target size while preserving aspect ratio.

    Returns:
        padded_image: uint8 array of shape (target_size, target_size, 3).
        scale: scale factor applied to the original image dimensions.
        pad: (pad_x, pad_y) applied after scaling.
    """
    h, w = image.shape[:2]
    scale = target_size / max(h, w)
    new_w, new_h = int(round(w * scale)), int(round(h * scale))

    resized = np.array(
        Image.fromarray(image).resize((new_w, new_h), Image.Resampling.BILINEAR),
        dtype=np.uint8,
    )
    padded = np.full((target_size, target_size, 3), pad_color, dtype=np.uint8)

    pad_x = (target_size - new_w) // 2
    pad_y = (target_size - new_h) // 2
    padded[pad_y : pad_y + new_h, pad_x : pad_x + new_w] = resized

    return padded, scale, (float(pad_x), float(pad_y))


def _preprocess(image: np.ndarray) -> tuple[np.ndarray, float, tuple[float, float]]:
    """Convert RGB image to normalized CHW tensor."""
    padded, scale, pad = _letterbox(image, target_size=INPUT_SIZE)
    tensor = padded.astype(np.float32) / 255.0
    tensor = np.transpose(tensor, (2, 0, 1))
    tensor = np.expand_dims(tensor, axis=0)
    return tensor, scale, pad


def nms(
    boxes: np.ndarray,
    scores: np.ndarray,
    iou_threshold: float = DEFAULT_IOU,
    max_detections: int = 300,
) -> list[int]:
    """Greedy non-maximum suppression.

    Args:
        boxes: Array of shape (N, 4) in [x1, y1, x2, y2] format.
        scores: Array of shape (N,) with confidence scores.
        iou_threshold: IoU threshold for suppression.
        max_detections: Maximum number of detections to return.

    Returns:
        Indices of boxes kept after NMS.
    """
    if len(boxes) == 0:
        return []

    x1 = boxes[:, 0]
    y1 = boxes[:, 1]
    x2 = boxes[:, 2]
    y2 = boxes[:, 3]
    areas = (x2 - x1) * (y2 - y1)

    order = scores.argsort()[::-1]
    keep: list[int] = []

    while order.size > 0 and len(keep) < max_detections:
        i = int(order[0])
        keep.append(i)

        xx1 = np.maximum(x1[i], x1[order[1:]])
        yy1 = np.maximum(y1[i], y1[order[1:]])
        xx2 = np.minimum(x2[i], x2[order[1:]])
        yy2 = np.minimum(y2[i], y2[order[1:]])

        w = np.maximum(0.0, xx2 - xx1)
        h = np.maximum(0.0, yy2 - yy1)
        inter = w * h
        union = areas[i] + areas[order[1:]] - inter
        iou = np.divide(inter, union, out=np.zeros_like(inter), where=union != 0)

        order = order[1:][iou <= iou_threshold]

    return keep


class YoloDetector:
    """Loads a YOLOv8 ONNX model and runs person detection on CPU."""

    def __init__(
        self,
        model_path: str | Path,
        confidence_threshold: float = DEFAULT_CONFIDENCE,
        iou_threshold: float = DEFAULT_IOU,
        execution_providers: Sequence[str] | None = None,
    ):
        model_path = Path(model_path)
        if not model_path.exists():
            raise FileNotFoundError(f"ONNX model not found: {model_path}")

        self.model_path = model_path
        self.confidence_threshold = confidence_threshold
        self.iou_threshold = iou_threshold
        self.execution_providers = list(execution_providers or ["CPUExecutionProvider"])

        self._session = ort.InferenceSession(
            str(model_path),
            providers=self.execution_providers,
        )
        self.input_name = self._session.get_inputs()[0].name
        logger.info(
            "Loaded ONNX model %s on providers %s",
            model_path,
            self._session.get_providers(),
        )

    def __call__(self, image: np.ndarray) -> list[Detection]:
        """Run inference on an RGB image and return person detections."""
        tensor, scale, pad = _preprocess(image)
        outputs = self._session.run(None, {self.input_name: tensor})
        return self._postprocess(outputs[0], scale, pad, image.shape[:2])

    def _postprocess(
        self,
        output: np.ndarray,
        scale: float,
        pad: tuple[float, float],
        original_shape: tuple[int, int],
    ) -> list[Detection]:
        """Decode YOLOv8 output and run NMS.

        Supports both (1, 84, N) and (1, N, 84) output layouts.
        """
        predictions = np.squeeze(output)
        if predictions.ndim != 2:
            raise ValueError(f"Unexpected model output shape: {output.shape}")

        # YOLOv8 outputs (84, N); transpose to (N, 84) for row-wise detections.
        if predictions.shape[0] == 84:
            predictions = predictions.T
        elif predictions.shape[1] != 84:
            raise ValueError(f"Unexpected model output shape: {output.shape}")

        class_scores = predictions[:, 4:]
        class_ids = np.argmax(class_scores, axis=1)
        confidences = np.max(class_scores, axis=1)

        mask = (confidences >= self.confidence_threshold) & (class_ids == PERSON_CLASS_ID)
        filtered = predictions[mask]
        confidences = confidences[mask]

        if len(filtered) == 0:
            return []

        xywh = filtered[:, :4]
        xyxy = np.zeros_like(xywh)
        xyxy[:, 0] = xywh[:, 0] - xywh[:, 2] / 2
        xyxy[:, 1] = xywh[:, 1] - xywh[:, 3] / 2
        xyxy[:, 2] = xywh[:, 0] + xywh[:, 2] / 2
        xyxy[:, 3] = xywh[:, 1] + xywh[:, 3] / 2

        pad_x, pad_y = pad
        orig_h, orig_w = original_shape
        xyxy[:, [0, 2]] = (xyxy[:, [0, 2]] - pad_x) / scale
        xyxy[:, [1, 3]] = (xyxy[:, [1, 3]] - pad_y) / scale

        xyxy[:, [0, 2]] = np.clip(xyxy[:, [0, 2]], 0, orig_w)
        xyxy[:, [1, 3]] = np.clip(xyxy[:, [1, 3]], 0, orig_h)

        keep = nms(xyxy, confidences, iou_threshold=self.iou_threshold)

        detections: list[Detection] = []
        for idx in keep:
            x1, y1, x2, y2 = xyxy[idx]
            detections.append(
                Detection(
                    label="person",
                    confidence=float(confidences[idx]),
                    box=BoundingBox(
                        x1=float(x1) / orig_w,
                        y1=float(y1) / orig_h,
                        x2=float(x2) / orig_w,
                        y2=float(y2) / orig_h,
                    ),
                )
            )

        return detections


class QueueDepthEstimator:
    """High-level estimator combining a YOLO detector with ROI-based counting."""

    def __init__(
        self,
        model_path: str | Path,
        confidence_threshold: float = DEFAULT_CONFIDENCE,
        iou_threshold: float = DEFAULT_IOU,
        seconds_per_person: int = SECONDS_PER_PERSON,
    ):
        self.detector = YoloDetector(
            model_path=model_path,
            confidence_threshold=confidence_threshold,
            iou_threshold=iou_threshold,
        )
        self.confidence_threshold = confidence_threshold
        self.seconds_per_person = seconds_per_person

    def estimate(self, image_bytes: bytes) -> QueueDepthResult:
        """Estimate queue depth from raw camera frame bytes."""
        image = decode_image(image_bytes)
        detections = self.detector(image)
        people = [
            d for d in detections
            if d.label.lower() == "person" and d.confidence >= self.confidence_threshold
        ]
        queue_depth = len(people)
        return QueueDepthResult(
            queue_depth=queue_depth,
            total_people=queue_depth,
            people_in_roi=queue_depth,
            people_outside_roi=0,
            estimated_wait_seconds=queue_depth * self.seconds_per_person,
            detections=people,
        )

    @property
    def model_path(self) -> Path:
        return self.detector.model_path
