from __future__ import annotations

from collections import Counter
from pathlib import Path
from typing import Any
from xml.etree import ElementTree
import io
import zipfile

from PIL import Image
from pypdf import PdfReader


MAX_EXCERPT_CHARS = 1200
MAX_ZIP_SAMPLE_ENTRIES = 50


class ParseError(ValueError):
    pass


def parse_artifact_file(storage_path_or_url: str, artifact_kind: str) -> tuple[str, dict[str, Any], list[dict[str, Any]], list[dict[str, Any]]]:
    path = resolve_local_path(storage_path_or_url)
    extension = path.suffix.lower()
    metadata: dict[str, Any] = {
        "artifact_kind": artifact_kind,
        "extension": extension,
        "path": str(path),
    }
    if extension in {".txt", ".md"}:
        excerpt = parse_text(path)
        metadata["parser"] = "text_excerpt"
        metadata["excerpt_chars"] = len(excerpt)
        return excerpt, metadata, [], []
    if extension == ".docx":
        excerpt, summary = parse_docx(path)
        metadata.update(summary)
        metadata["parser"] = "docx_text_excerpt"
        return excerpt, metadata, [], []
    if extension == ".pdf":
        excerpt, summary = parse_pdf(path)
        metadata.update(summary)
        metadata["parser"] = "pdf_text_excerpt"
        return excerpt, metadata, [], []
    if extension in {".png", ".jpg", ".jpeg"}:
        summary = parse_image(path)
        metadata.update(summary)
        metadata["parser"] = "image_metadata"
        return "", metadata, [], []
    if extension == ".zip":
        summary = parse_zip(path)
        metadata.update(summary)
        metadata["parser"] = "zip_manifest"
        return "", metadata, [], []
    raise ParseError(f"unsupported extension: {extension or '(none)'}")


def resolve_local_path(storage_path_or_url: str) -> Path:
    raw = storage_path_or_url.strip()
    if raw.startswith("http://") or raw.startswith("https://"):
        raise ParseError("remote URLs are not supported in the current middleware phase")
    path = Path(raw)
    if not path.exists() or not path.is_file():
        raise ParseError(f"artifact path does not exist: {raw}")
    return path


def parse_text(path: Path) -> str:
    content = path.read_text(encoding="utf-8", errors="replace")
    return sanitize_excerpt(content)


def parse_docx(path: Path) -> tuple[str, dict[str, Any]]:
    with zipfile.ZipFile(path) as archive:
        try:
            document_xml = archive.read("word/document.xml")
        except KeyError as exc:
            raise ParseError("docx document.xml is unavailable") from exc
        file_names = archive.namelist()
    root = ElementTree.fromstring(document_xml)
    text_parts: list[str] = []
    paragraph_count = 0
    table_count = 0
    for node in root.iter():
        if node.tag.endswith("}p"):
            paragraph_count += 1
        elif node.tag.endswith("}tbl"):
            table_count += 1
        elif node.tag.endswith("}t") and node.text:
            text_parts.append(node.text.strip())
    excerpt = sanitize_excerpt(" ".join(part for part in text_parts if part))
    summary = {
        "paragraph_count": paragraph_count,
        "table_count": table_count,
        "file_count": len(file_names),
        "excerpt_chars": len(excerpt),
    }
    return excerpt, summary


def parse_pdf(path: Path) -> tuple[str, dict[str, Any]]:
    reader = PdfReader(str(path))
    page_count = len(reader.pages)
    texts: list[str] = []
    processed_pages = 0
    for page in reader.pages[:5]:
        text = page.extract_text() or ""
        text = text.strip()
        if text:
            texts.append(text)
        processed_pages += 1
    excerpt = sanitize_excerpt("\n".join(texts))
    summary = {
        "page_count": page_count,
        "pages_processed": processed_pages,
        "excerpt_chars": len(excerpt),
    }
    return excerpt, summary


def parse_image(path: Path) -> dict[str, Any]:
    with Image.open(path) as image:
        width, height = image.size
        mode = image.mode
    return {
        "width": width,
        "height": height,
        "mode": mode,
    }


def parse_zip(path: Path) -> dict[str, Any]:
    with zipfile.ZipFile(path) as archive:
        infos = archive.infolist()
        sample_entries = sorted(info.filename for info in infos[:MAX_ZIP_SAMPLE_ENTRIES])
        extension_counts = Counter()
        total_uncompressed = 0
        for info in infos:
            suffix = Path(info.filename).suffix.lower() or "(none)"
            extension_counts[suffix] += 1
            total_uncompressed += info.file_size
    return {
        "file_count": len(infos),
        "uncompressed_bytes": total_uncompressed,
        "sample_entries": sample_entries,
        "extension_counts": dict(extension_counts),
    }


def sanitize_excerpt(text: str) -> str:
    collapsed = " ".join(text.split())
    if len(collapsed) <= MAX_EXCERPT_CHARS:
        return collapsed
    return collapsed[: MAX_EXCERPT_CHARS - 3] + "..."
