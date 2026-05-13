from __future__ import annotations

import re
import threading
import time
import uuid
from collections import Counter
from typing import Any

from .models import BuildRetrievalIndexRequest, QueryRetrievalRequest

MAX_INDEXES = 128
DEFAULT_CHUNK_SIZE = 320
DEFAULT_CHUNK_OVERLAP = 40


class RetrievalError(RuntimeError):
    pass


class RetrievalStore:
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._entries: dict[str, dict[str, Any]] = {}

    def build_index(self, request: BuildRetrievalIndexRequest) -> tuple[str, int]:
        chunks = normalize_chunks(request.chunks)
        if len(chunks) == 0:
            raise RetrievalError("retrieval index requires at least one non-empty chunk")
        index_ref = f"idx:{request.submission_id}:{uuid.uuid4().hex[:12]}"
        entry = {
            "created_at": time.time(),
            "submission_id": request.submission_id,
            "artifact_ids": list(request.artifact_ids),
            "chunks": chunks,
        }
        with self._lock:
            self._entries[index_ref] = entry
            self._trim_locked()
        return index_ref, len(chunks)

    def query_index(self, request: QueryRetrievalRequest) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
        with self._lock:
            entry = self._entries.get(request.index_ref)
        if entry is None:
            raise RetrievalError(f"retrieval index {request.index_ref!r} was not found")
        query_tokens = tokenize(request.query)
        if len(query_tokens) == 0:
            raise RetrievalError("retrieval query must contain at least one token")

        scored: list[dict[str, Any]] = []
        for chunk in entry["chunks"]:
            overlap = chunk["token_counter"] & Counter(query_tokens)
            score = sum(overlap.values())
            if score <= 0:
                continue
            scored.append(
                {
                    "chunk_id": chunk["chunk_id"],
                    "artifact_id": chunk["artifact_id"],
                    "score": score,
                    "text": chunk["text"],
                    "evidence_ref": chunk["evidence_ref"],
                    "keywords": sorted(overlap.keys()),
                }
            )
        scored.sort(key=lambda item: (-item["score"], item["chunk_id"]))
        limited = scored[: max(1, min(request.top_k, 20))]
        citations = [
            {
                "chunk_id": item["chunk_id"],
                "artifact_id": item["artifact_id"],
                "evidence_ref": item["evidence_ref"],
            }
            for item in limited
        ]
        return limited, citations

    def _trim_locked(self) -> None:
        if len(self._entries) <= MAX_INDEXES:
            return
        ordered = sorted(self._entries.items(), key=lambda item: item[1]["created_at"])
        for index_ref, _entry in ordered[: len(self._entries) - MAX_INDEXES]:
            self._entries.pop(index_ref, None)


STORE = RetrievalStore()


def chunk_artifact_text(
    artifact_id: str,
    evidence_ref: str,
    text: str,
    *,
    chunk_size: int = DEFAULT_CHUNK_SIZE,
    overlap: int = DEFAULT_CHUNK_OVERLAP,
) -> list[dict[str, Any]]:
    normalized = normalize_text(text)
    if normalized == "":
        return []
    words = normalized.split()
    if not words:
        return []
    chunk_size = max(80, chunk_size)
    overlap = max(0, min(overlap, chunk_size // 2))
    step = max(1, chunk_size - overlap)
    chunks: list[dict[str, Any]] = []
    for start in range(0, len(words), step):
        window = words[start : start + chunk_size]
        if not window:
            break
        chunk_text = " ".join(window).strip()
        if chunk_text == "":
            continue
        chunks.append(
            {
                "chunk_id": f"{artifact_id}:{len(chunks)+1}",
                "artifact_id": artifact_id,
                "evidence_ref": evidence_ref,
                "text": chunk_text,
                "token_counter": Counter(tokenize(chunk_text)),
            }
        )
        if start + chunk_size >= len(words):
            break
    return chunks


def normalize_chunks(raw_chunks: list[dict[str, Any]]) -> list[dict[str, Any]]:
    chunks: list[dict[str, Any]] = []
    for index, chunk in enumerate(raw_chunks, start=1):
        if not isinstance(chunk, dict):
            continue
        text = normalize_text(chunk.get("text", ""))
        if text == "":
            continue
        artifact_id = str(chunk.get("artifact_id", "")).strip() or f"artifact-{index}"
        evidence_ref = str(chunk.get("evidence_ref", "")).strip() or f"artifact:{artifact_id}"
        chunk_id = str(chunk.get("chunk_id", "")).strip() or f"{artifact_id}:{index}"
        chunks.append(
            {
                "chunk_id": chunk_id,
                "artifact_id": artifact_id,
                "evidence_ref": evidence_ref,
                "text": text,
                "token_counter": Counter(tokenize(text)),
            }
        )
    return chunks


def tokenize(text: str) -> list[str]:
    return [token for token in re.findall(r"[a-zA-Z0-9_\-\u4e00-\u9fff]+", text.lower()) if token]


def normalize_text(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, bytes):
        text = value.decode("utf-8", errors="ignore")
    elif isinstance(value, str):
        text = value
    else:
        return ""
    text = text.replace("\r", " ").replace("\n", " ")
    text = re.sub(r"\s+", " ", text)
    return text.strip()
