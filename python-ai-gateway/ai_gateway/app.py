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
from .evaluator import EvaluationError, evaluate_submission_request
from .parser import ParseError, parse_artifact_file
from .retrieval import RetrievalError, STORE


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
    try:
        return evaluate_submission_request(request)
    except EvaluationError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc


@app.post("/internal/build-retrieval-index", response_model=BuildRetrievalIndexResponse)
def build_retrieval_index(
    request: BuildRetrievalIndexRequest,
    authorization: str | None = Header(default=None),
) -> BuildRetrievalIndexResponse:
    maybe_require_auth(authorization)
    try:
        index_ref, chunk_count = STORE.build_index(request)
    except RetrievalError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    return BuildRetrievalIndexResponse(index_ref=index_ref, chunk_count=chunk_count)


@app.post("/internal/query-retrieval", response_model=QueryRetrievalResponse)
def query_retrieval(
    request: QueryRetrievalRequest,
    authorization: str | None = Header(default=None),
) -> QueryRetrievalResponse:
    maybe_require_auth(authorization)
    try:
        matches, citations = STORE.query_index(request)
    except RetrievalError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    return QueryRetrievalResponse(matches=matches, citations=citations)
