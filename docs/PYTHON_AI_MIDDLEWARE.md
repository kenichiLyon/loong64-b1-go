# Python AI Middleware

This document describes the internal Python service used for AI-heavy capabilities.

## Purpose

The Python middleware exists to isolate features that are expensive to build and maintain in Go:

- document parsing
- OCR
- embeddings
- retrieval
- local model adapters
- structured AI evaluation payload generation

## Current Phase

The current phase delivers:

- service skeleton
- stable internal request / response models
- real artifact parsing for:
  - `txt`
  - `md`
  - `docx`
  - `pdf`
  - `png`
  - `jpg` / `jpeg`
  - `zip`
- Go-side readiness integration and internal HTTP client

The current phase does not yet deliver:

- OCR
- embedding generation
- vector index persistence
- retrieval execution against a real vector store
- local model inference
- production routing from Go teaching workflows into the middleware

## Service Contract

Internal endpoints:

- `GET /health/live`
- `GET /health/ready`
- `POST /internal/parse-artifact`
- `POST /internal/evaluate-submission`
- `POST /internal/build-retrieval-index`
- `POST /internal/query-retrieval`

The Go service remains the public API and persists all business state.

## Local Run

```bash
cd python-ai-gateway
python -m venv .venv
. .venv/bin/activate
pip install -e .
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

Windows PowerShell:

```powershell
cd python-ai-gateway
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -e .
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

## Go Integration

Set:

```env
AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_TIMEOUT=10s
AI_GATEWAY_API_KEY=
```

When `AI_GATEWAY_BASE_URL` is set, the Go service includes the middleware in readiness checks.

## Design Notes

- The middleware does not write the business database directly.
- The middleware does not decide whether a score is published.
- The middleware only returns structured internal payloads to the Go service.
- LangChain is intentionally not used in this phase.
