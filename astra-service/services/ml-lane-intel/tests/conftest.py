"""Shared pytest configuration and fixtures."""

from __future__ import annotations

from pathlib import Path

import pytest

DEFAULT_MODEL_PATH = Path("models/yolov8n.onnx")


def _build_or_locate_model(tmp_path: Path) -> Path:
    """Return a usable ONNX model path, building a stub if possible."""
    try:
        import models.download_model as download_model

        path = tmp_path / "yolov8n.onnx"
        download_model.main(["--output", str(path), "--force"])
        return path
    except Exception as exc:  # pragma: no cover
        if DEFAULT_MODEL_PATH.exists():
            return DEFAULT_MODEL_PATH
        raise pytest.skip(
            f"ONNX model unavailable and stub could not be built ({exc}); skipping tests."
        ) from exc


@pytest.fixture(scope="session")
def model_path(tmp_path_factory: pytest.TempPathFactory) -> Path:
    """Generate the stub ONNX model once per test session."""
    tmp_path = tmp_path_factory.mktemp("models")
    return _build_or_locate_model(tmp_path)


@pytest.fixture(scope="session")
def fake_image_bytes() -> bytes:
    """Create a small blank JPEG image in memory."""
    from io import BytesIO

    from PIL import Image

    image = Image.new("RGB", (640, 480), color=(0, 0, 0))
    buffer = BytesIO()
    image.save(buffer, format="JPEG")
    return buffer.getvalue()
