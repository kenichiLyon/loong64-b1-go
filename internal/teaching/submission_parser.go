package teaching

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxArchiveFiles             = 1000
	maxArchiveUncompressedBytes = 200 * 1024 * 1024
	maxTextExcerptBytes         = 4096
)

var commitSHAPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)

type storedArtifact struct {
	Kind        ArtifactKind
	ContentType string
	ByteSize    int64
	SHA256Hex   string
	StorageKey  string
	Metadata    json.RawMessage
	TextExcerpt string
}

type prefixBuffer struct {
	buf   []byte
	limit int
}

func (p *prefixBuffer) Write(data []byte) (int, error) {
	if len(p.buf) < p.limit {
		remaining := p.limit - len(p.buf)
		if remaining > len(data) {
			remaining = len(data)
		}
		p.buf = append(p.buf, data[:remaining]...)
	}
	return len(data), nil
}

func (s *Service) storeUploadedArtifact(input ArtifactUploadInput, artifactID, submissionID string) (storedArtifact, error) {
	if s.store == nil {
		return storedArtifact{}, unavailableError("artifact storage is not configured", nil)
	}
	if input.Reader == nil {
		return storedArtifact{}, validationError("file is required")
	}
	kind, ext, err := resolveArtifactKind(input.FileName, input.DeclaredKind)
	if err != nil {
		return storedArtifact{}, err
	}
	fileName := safeFileName(input.FileName, ext)
	storageKey := path.Join("artifacts", submissionID, artifactID, fileName)
	targetPath, err := s.store.Resolve(storageKey)
	if err != nil {
		return storedArtifact{}, unavailableError("resolve artifact storage key", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
		return storedArtifact{}, unavailableError("create artifact storage directory", err)
	}
	file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return storedArtifact{}, unavailableError("create artifact object", err)
	}
	keepFile := false
	defer func() {
		_ = file.Close()
		if !keepFile {
			_ = os.Remove(targetPath)
		}
	}()

	hasher := sha256.New()
	prefix := &prefixBuffer{limit: 512}
	limit := &io.LimitedReader{R: input.Reader, N: s.maxUploadBytes + 1}
	written, err := io.Copy(io.MultiWriter(file, hasher, prefix), limit)
	if err != nil {
		return storedArtifact{}, unavailableError("store artifact object", err)
	}
	if written > s.maxUploadBytes {
		return storedArtifact{}, validationError(fmt.Sprintf("file exceeds max upload size of %d bytes", s.maxUploadBytes))
	}
	if written == 0 {
		return storedArtifact{}, validationError("file must not be empty")
	}
	if err := file.Close(); err != nil {
		return storedArtifact{}, unavailableError("close artifact object", err)
	}
	detectedType := http.DetectContentType(prefix.buf)
	if err := validateContentType(ext, detectedType); err != nil {
		return storedArtifact{}, err
	}
	metadata, excerpt, err := analyzeStoredArtifact(targetPath, kind, ext, detectedType)
	if err != nil {
		return storedArtifact{}, err
	}
	keepFile = true
	return storedArtifact{
		Kind:        kind,
		ContentType: detectedType,
		ByteSize:    written,
		SHA256Hex:   hex.EncodeToString(hasher.Sum(nil)),
		StorageKey:  storageKey,
		Metadata:    metadata,
		TextExcerpt: excerpt,
	}, nil
}

func resolveArtifactKind(fileName, declared string) (ArtifactKind, string, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return "", "", validationError("file extension is required")
	}
	inferred, ok := kindByExtension(ext)
	if !ok {
		return "", "", validationError("unsupported file extension: " + ext)
	}
	declared = strings.ToLower(strings.TrimSpace(declared))
	if declared == "" {
		return inferred, ext, nil
	}
	kind := ArtifactKind(declared)
	switch kind {
	case ArtifactKindDocument, ArtifactKindReport, ArtifactKindScreenshot, ArtifactKindCodeArchive, ArtifactKindOther:
	default:
		return "", "", validationError("invalid artifact_kind")
	}
	if !kindCompatibleWithExtension(kind, ext) {
		return "", "", validationError("artifact_kind does not match file extension")
	}
	return kind, ext, nil
}

func kindByExtension(ext string) (ArtifactKind, bool) {
	switch ext {
	case ".doc", ".docx", ".pdf":
		return ArtifactKindDocument, true
	case ".md", ".txt":
		return ArtifactKindReport, true
	case ".png", ".jpg", ".jpeg":
		return ArtifactKindScreenshot, true
	case ".zip":
		return ArtifactKindCodeArchive, true
	default:
		return "", false
	}
}

func kindCompatibleWithExtension(kind ArtifactKind, ext string) bool {
	switch kind {
	case ArtifactKindDocument:
		return ext == ".doc" || ext == ".docx" || ext == ".pdf" || ext == ".txt" || ext == ".md"
	case ArtifactKindReport:
		return ext == ".doc" || ext == ".docx" || ext == ".pdf" || ext == ".txt" || ext == ".md"
	case ArtifactKindScreenshot:
		return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
	case ArtifactKindCodeArchive:
		return ext == ".zip"
	case ArtifactKindOther:
		return true
	default:
		return false
	}
}

func safeFileName(fileName, ext string) string {
	base := filepath.Base(strings.TrimSpace(fileName))
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.Trim(base, ". ")
	if base == "" || base == "." {
		base = "artifact" + ext
	}
	var b strings.Builder
	for _, r := range base {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	cleaned := b.String()
	if cleaned == "" {
		return "artifact" + ext
	}
	if len(cleaned) > 120 {
		cleaned = cleaned[:120]
	}
	return cleaned
}

func validateContentType(ext, detected string) error {
	switch ext {
	case ".png":
		if detected != "image/png" {
			return validationError("PNG screenshot content type mismatch")
		}
	case ".jpg", ".jpeg":
		if detected != "image/jpeg" {
			return validationError("JPEG screenshot content type mismatch")
		}
	case ".pdf":
		if detected != "application/pdf" {
			return validationError("PDF content type mismatch")
		}
	case ".zip", ".docx":
		if detected != "application/zip" && detected != "application/octet-stream" {
			return validationError("ZIP-based artifact content type mismatch")
		}
	case ".txt", ".md":
		if !strings.HasPrefix(detected, "text/plain") && detected != "application/octet-stream" {
			return validationError("text report content type mismatch")
		}
	case ".doc":
		if detected == "application/x-msdownload" {
			return validationError("legacy Word artifact content type mismatch")
		}
	}
	return nil
}

func analyzeStoredArtifact(filePath string, kind ArtifactKind, ext, contentType string) (json.RawMessage, string, error) {
	metadata := map[string]any{
		"artifact_kind": string(kind),
		"extension":     ext,
		"content_type":  contentType,
		"parser":        "metadata_only",
	}
	var excerpt string
	switch ext {
	case ".txt", ".md":
		text, err := readTextExcerpt(filePath)
		if err != nil {
			return nil, "", err
		}
		excerpt = text
		metadata["parser"] = "text_excerpt"
		metadata["excerpt_bytes"] = len(text)
	case ".png", ".jpg", ".jpeg":
		width, height, err := imageDimensions(filePath)
		if err != nil {
			return nil, "", validationError("invalid screenshot image")
		}
		metadata["width"] = width
		metadata["height"] = height
		metadata["parser"] = "image_metadata"
	case ".zip":
		summary, err := inspectZip(filePath)
		if err != nil {
			return nil, "", err
		}
		metadata["parser"] = "zip_manifest"
		for key, value := range summary {
			metadata[key] = value
		}
	case ".docx":
		summary, err := inspectZip(filePath)
		if err != nil {
			return nil, "", err
		}
		metadata["parser"] = "docx_container_metadata"
		for key, value := range summary {
			metadata[key] = value
		}
	case ".pdf":
		metadata["parser"] = "pdf_metadata_stub"
		metadata["note"] = "deep PDF text extraction is deferred to the parsing worker"
	case ".doc":
		metadata["parser"] = "word_legacy_metadata_stub"
		metadata["note"] = "legacy Word extraction is deferred to manual review or a target-safe parser"
	}
	return mustJSON(metadata), excerpt, nil
}

func readTextExcerpt(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", unavailableError("open text artifact", err)
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, maxTextExcerptBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", unavailableError("read text artifact", err)
	}
	if !utf8.Valid(data) {
		return strings.ToValidUTF8(string(data), ""), nil
	}
	return string(bytes.TrimSpace(data)), nil
}

func imageDimensions(filePath string) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, unavailableError("open screenshot artifact", err)
	}
	defer func() { _ = file.Close() }()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func inspectZip(filePath string) (map[string]any, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, validationError("invalid zip container")
	}
	defer func() { _ = reader.Close() }()
	if len(reader.File) > maxArchiveFiles {
		return nil, validationError("archive contains too many files")
	}
	var total uint64
	entries := make([]string, 0, min(len(reader.File), 50))
	extCounts := map[string]int{}
	for _, file := range reader.File {
		if err := validateArchiveEntry(file); err != nil {
			return nil, err
		}
		total += file.UncompressedSize64
		if total > maxArchiveUncompressedBytes {
			return nil, validationError("archive uncompressed size exceeds safety limit")
		}
		if !file.FileInfo().IsDir() && len(entries) < 50 {
			entries = append(entries, file.Name)
		}
		ext := strings.ToLower(path.Ext(file.Name))
		if ext == "" {
			ext = "(none)"
		}
		extCounts[ext]++
	}
	sort.Strings(entries)
	return map[string]any{
		"file_count":         len(reader.File),
		"uncompressed_bytes": total,
		"sample_entries":     entries,
		"extension_counts":   extCounts,
	}, nil
}

func validateArchiveEntry(file *zip.File) error {
	name := strings.TrimSpace(file.Name)
	if name == "" {
		return validationError("archive contains an empty path")
	}
	if strings.Contains(name, "\\") || strings.Contains(name, ":") {
		return validationError("archive contains an unsafe path")
	}
	cleaned := path.Clean(name)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." || path.IsAbs(cleaned) {
		return validationError("archive contains a path traversal entry")
	}
	if file.FileInfo().Mode()&os.ModeSymlink != 0 {
		return validationError("archive contains a symbolic link")
	}
	return nil
}

func validateGitLink(input CreateGitLinkInput) (json.RawMessage, error) {
	rawURL := strings.TrimSpace(input.URL)
	if rawURL == "" {
		return nil, validationError("url is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return nil, validationError("url must be absolute")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, validationError("url must use http or https")
	}
	if parsed.User != nil {
		return nil, validationError("url must not contain credentials")
	}
	commitSHA := strings.TrimSpace(input.CommitSHA)
	if commitSHA != "" && !commitSHAPattern.MatchString(commitSHA) {
		return nil, validationError("commit_sha must be a 7-64 character hexadecimal value")
	}
	return mustJSON(map[string]any{
		"parser":     "git_link_metadata",
		"url_host":   strings.ToLower(parsed.Host),
		"commit_sha": commitSHA,
		"note":       strings.TrimSpace(input.Note),
		"fetch":      "deferred",
	}), nil
}

func mustJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
