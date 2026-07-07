"""Unit tests for the ONNX/YOLO model wrapper and queue-depth estimator."""

from __future__ import annotations

from pathlib import Path

import numpy as np
import pytest
from PIL import Image

import models.download_model as download_model
from model import (
    BoundingBox,
    QueueDepthEstimator,
    QueueDepthResult,
    YoloDetector,
    decode_image,
    nms,
)


@pytest.fixture
def stub_model_path(tmp_path: Path) -> Path:
    path = tmp_path / "yolov8n.onnx"
    download_model.main(["--output", str(path), "--force"])
    return path


@pytest.fixture
def blank_jpeg() -> bytes:
    from io import BytesIO

    from PIL import Image

    image = Image.new("RGB", (640, 480), color=(0, 0, 0))
    buffer = BytesIO()
    image.save(buffer, format="JPEG")
    return buffer.getvalue()


class TestDecodeImage:
    def test_decodes_jpeg_bytes(self, blank_jpeg: bytes) -> None:
        image = decode_image(blank_jpeg)
        assert image.shape == (480, 640, 3)
        assert image.dtype == np.uint8


class TestNms:
    def test_removes_overlapping_boxes(self) -> None:
        boxes = np.array(
            [
                [0.0, 0.0, 1.0, 1.0],
                [0.1, 0.1, 1.1, 1.1],
                [2.0, 2.0, 3.0, 3.0],
            ],
            dtype=np.float32,
        )
        scores = np.array([0.9, 0.8, 0.7], dtype=np.float32)
        keep = nms(boxes, scores, iou_threshold=0.5)
        assert keep == [0, 2]

    def test_empty(self) -> None:
        keep = nms(np.zeros((0, 4)), np.zeros(0))
        assert keep == []

    def test_respects_max_detections(self) -> None:
        boxes = np.array(
            [
                [0.0, 0.0, 1.0, 1.0],
                [2.0, 2.0, 3.0, 3.0],
                [4.0, 4.0, 5.0, 5.0],
            ],
            dtype=np.float32,
        )
        scores = np.array([0.9, 0.8, 0.7], dtype=np.float32)
        keep = nms(boxes, scores, max_detections=2)
        assert keep == [0, 1]


class TestYoloDetector:
    def test_loads_stub_model(self, stub_model_path: Path) -> None:
        detector = YoloDetector(stub_model_path)
        assert detector.model_path == stub_model_path

    def test_detects_three_people_on_blank_frame(self, stub_model_path: Path, blank_jpeg: bytes) -> None:
        detector = YoloDetector(stub_model_path)
        image = decode_image(blank_jpeg)
        detections = detector(image)
        assert len(detections) == 3
        assert all(d.label == "person" for d in detections)
        assert all(d.confidence >= 0.25 for d in detections)

    def test_model_missing_raises_file_not_found(self) -> None:
        with pytest.raises(FileNotFoundError):
            YoloDetector(Path("models/does-not-exist.onnx"))


class TestQueueDepthEstimator:
    def test_estimate_from_bytes(self, stub_model_path: Path, blank_jpeg: bytes) -> None:
        estimator = QueueDepthEstimator(stub_model_path)
        result = estimator.estimate(blank_jpeg)
        assert isinstance(result, QueueDepthResult)
        assert result.queue_depth == 3
        assert result.total_people == 3
        assert result.people_in_roi == 3
        assert result.people_outside_roi == 0
        assert result.estimated_wait_seconds == 135
        assert len(result.detections) == 3

    def test_bounding_boxes_are_normalized(self, stub_model_path: Path, blank_jpeg: bytes) -> None:
        estimator = QueueDepthEstimator(stub_model_path)
        result = estimator.estimate(blank_jpeg)
        for det in result.detections:
            assert 0.0 <= det.box.x1 <= 1.0
            assert 0.0 <= det.box.y1 <= 1.0
            assert 0.0 <= det.box.x2 <= 1.0
            assert 0.0 <= det.box.y2 <= 1.0

    def test_empty_detections_have_zero_wait(self) -> None:
        result = QueueDepthResult(
            queue_depth=0,
            total_people=0,
            people_in_roi=0,
            people_outside_roi=0,
            estimated_wait_seconds=0,
            detections=[],
        )
        assert result.estimated_wait_seconds == 0


class TestBoundingBox:
    def test_area_and_center(self) -> None:
        box = BoundingBox(0.1, 0.2, 0.5, 0.8)
        assert box.width == pytest.approx(0.4)
        assert box.height == pytest.approx(0.6)
        assert box.area == pytest.approx(0.24)
        assert box.center == pytest.approx((0.3, 0.5))
