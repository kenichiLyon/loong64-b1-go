from __future__ import annotations

import os

from fastapi import FastAPI, Header, HTTPException

from .models import (
    BuildRetrievalIndexRequest,
    BuildRetrievalIndexResponse,
    EvaluateSubmissionRequest,
    EvaluateSubmissionResponse,
    HealthResponse,
    ParseArtifactRequest,
    ParseArtifactResponse,
    QueryRetrievalRequest,
    QueryRetrievalResponse,
)
from .parser import ParseError, parse_artifact_file


app = FastAPI(title="python-ai-gateway", version="0.1.0")


def maybe_require_auth(authorization: str | None) -> None:
    expected = os.getenv("AI_GATEWAY_API_KEY", "").strip()
    if expected == "":
        return
    if authorization is None or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="missing bearer token")
    provided = authorization.removeprefix("Bearer ").strip()
    if provided != expected:
        raise HTTPException(status_code=401, detail="invalid bearer token")


@app.get("/health/live", response_model=HealthResponse)
def live() -> HealthResponse:
    return HealthResponse()


@app.get("/health/ready", response_model=HealthResponse)
def ready() -> HealthResponse:
    return HealthResponse()


@app.post("/internal/parse-artifact", response_model=ParseArtifactResponse)
def parse_artifact(
    request: ParseArtifactRequest,
    authorization: str | None = Header(default=None),
) -> ParseArtifactResponse:
    maybe_require_auth(authorization)
    try:
        excerpt, metadata, sections, evidence = parse_artifact_file(
            storage_path_or_url=request.storage_path_or_url,
            artifact_kind=request.artifact_kind,
            content_type=request.content_type,
            parse_options=request.parse_options,
        )
    except ParseError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    metadata["artifact_id"] = request.artifact_id
    return ParseArtifactResponse(
        status="succeeded",
        text_excerpt=excerpt,
        metadata=metadata,
        sections=sections,
        evidence=evidence,
    )


@app.post("/internal/evaluate-submission", response_model=EvaluateSubmissionResponse)
def evaluate_submission(
    request: EvaluateSubmissionRequest,
    authorization: str | None = Header(default=None),
) -> EvaluateSubmissionResponse:
    maybe_require_auth(authorization)
    return EvaluateSubmissionResponse(
        summary="stub evaluation result",
        findings=[],
        metric_scores=[],
        confidence=0.0,
        raw_model_meta={"mode": request.mode, "engine": "stub"},
    )


@app.post("/internal/build-retrieval-index", response_model=BuildRetrievalIndexResponse)
def build_retrieval_index(
    request: BuildRetrievalIndexRequest,
    authorization: str | None = Header(default=None),
) -> BuildRetrievalIndexResponse:
    maybe_require_auth(authorization)
    return BuildRetrievalIndexResponse(
        index_ref=f"stub-index:{request.submission_id}",
        chunk_count=len(request.chunks),
    )


@app.post("/internal/query-retrieval", response_model=QueryRetrievalResponse)
def query_retrieval(
    request: QueryRetrievalRequest,
    authorization: str | None = Header(default=None),
) -> QueryRetrievalResponse:
    maybe_require_auth(authorization)
    return QueryRetrievalResponse(
        matches=[],
        citations=[],
    )
