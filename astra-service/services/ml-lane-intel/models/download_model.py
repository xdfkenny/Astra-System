"""Download or synthesize a YOLOv8-compatible ONNX model for local testing.

This script is used at build/test time only. By default it writes a tiny
deterministic stub model that emits three person boxes. The stub is fully
sufficient for unit tests and CI. Pass ``--real`` to download the official
YOLOv8n ONNX release from Ultralytics (requires an internet connection).
"""

from __future__ import annotations

import argparse
from pathlib import Path

import numpy as np

try:
    import onnx
    from onnx import TensorProto, helper
except ImportError as exc:  # pragma: no cover
    raise ImportError(
        "The 'onnx' package is required to build the stub model. "
        "Install it with: pip install onnx"
    ) from exc


NUM_ANCHORS = 100
OUTPUT_SHAPE = (1, 84, NUM_ANCHORS)
PERSON_CLASS_INDEX = 0


def _build_stub_output() -> np.ndarray:
    """Build a deterministic output tensor with three high-confidence people."""
    data = np.zeros(OUTPUT_SHAPE, dtype=np.float32)

    people = [
        (0.25, 0.25, 0.08, 0.16, 0.92),
        (0.50, 0.50, 0.10, 0.20, 0.88),
        (0.75, 0.75, 0.12, 0.24, 0.85),
    ]

    for idx, (cx, cy, w, h, conf) in enumerate(people):
        data[0, 0, idx] = cx
        data[0, 1, idx] = cy
        data[0, 2, idx] = w
        data[0, 3, idx] = h
        data[0, 4 + PERSON_CLASS_INDEX, idx] = conf

    rng = np.random.default_rng(42)
    for idx in range(len(people), NUM_ANCHORS):
        data[0, 4:, idx] = rng.uniform(0.0, 0.05, size=80)

    return data


def _build_stub_onnx_model() -> onnx.ModelProto:
    """Create a minimal ONNX model that ignores its input and returns a constant."""
    input_tensor = helper.make_tensor_value_info(
        "images",
        TensorProto.FLOAT,
        [1, 3, 640, 640],
    )
    output_tensor = helper.make_tensor_value_info(
        "output0",
        TensorProto.FLOAT,
        list(OUTPUT_SHAPE),
    )

    constant_tensor = helper.make_tensor(
        "detections",
        TensorProto.FLOAT,
        list(OUTPUT_SHAPE),
        _build_stub_output().flatten().tolist(),
    )

    node = helper.make_node(
        "Constant",
        inputs=[],
        outputs=["output0"],
        value=constant_tensor,
        name="constant_detections",
    )

    graph = helper.make_graph(
        nodes=[node],
        name="yolov8n_stub",
        inputs=[input_tensor],
        outputs=[output_tensor],
    )

    opset = helper.make_operatorsetid("", 17)
    model = helper.make_model(graph, opset_imports=[opset])
    model.ir_version = 8
    onnx.checker.check_model(model)
    return model


def _download_real_yolo(model_path: Path) -> None:
    """Download the official YOLOv8n ONNX model from Ultralytics."""
    import urllib.request

    url = "https://github.com/ultralytics/assets/releases/download/v8.3.0/yolov8n.onnx"
    print(f"Downloading {url} ...")
    urllib.request.urlretrieve(url, model_path)
    print(f"Saved {model_path} ({model_path.stat().st_size} bytes)")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Prepare a YOLOv8 ONNX model")
    parser.add_argument(
        "--output",
        "-o",
        type=Path,
        default=Path("models/yolov8n.onnx"),
        help="Destination path for the model",
    )
    parser.add_argument(
        "--real",
        action="store_true",
        help="Download the real YOLOv8n model (requires internet)",
    )
    parser.add_argument(
        "--force",
        "-f",
        action="store_true",
        help="Overwrite an existing model file",
    )
    args = parser.parse_args(argv)

    output_path: Path = args.output
    if output_path.exists() and not args.force:
        print(f"Model already exists at {output_path}; use --force to overwrite.")
        return 0

    output_path.parent.mkdir(parents=True, exist_ok=True)

    if args.real:
        _download_real_yolo(output_path)
    else:
        model = _build_stub_onnx_model()
        onnx.save(model, output_path)
        print(f"Saved stub model to {output_path} ({output_path.stat().st_size} bytes)")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
