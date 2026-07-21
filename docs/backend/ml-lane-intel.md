# ML Lane Intelligence

## Overview

The ML Lane Intelligence service (`services/ml-lane-intel/`) uses computer vision to estimate queue length from kiosk camera feeds, dynamically adjusting UI behavior based on store traffic.

## Technology

- **Language:** Python 3.12-3.13
- **Framework:** FastAPI / Uvicorn
- **ML Model:** YOLOv8n (ONNX format)
- **Dependencies:** `pyproject.toml` with uv package manager

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/lane/queue-depth` | Analyze camera frame for queue length |
| GET | `/v1/lane/stream` | SSE: streaming queue depth updates |

## How It Works

1. Camera captures frame
2. YOLOv8n model detects persons in frame
3. Estimated queue depth calculated
4. Results cached in Redis (TTL: 5s)
5. Kiosk UI adjusts mode based on traffic:
   - **Low traffic:** Full menu, detailed item views
   - **High traffic:** Express mode, streamlined interface

## Models

Models stored in `services/ml-lane-intel/models/` in ONNX format.
