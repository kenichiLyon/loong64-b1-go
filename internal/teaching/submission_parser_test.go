package teaching

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenichiLyon/loong64-b1-go/internal/storage"
	"github.com/phpdave11/gofpdf"
)

func TestInspectZipRejectsPathTraversal(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	createTestZip(t, zipPath, map[string]string{"../evil.txt": "nope"})
	if _, err := inspectZip(zipPath); ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation error for zip slip entry, got %v", err)
	}
}

func TestInspectZipRejectsSymlink(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "symlink.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: "link-to-secret", Method: zip.Store}
	header.SetMode(fs.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("secret.txt")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectZip(zipPath); ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation error for symlink entry, got %v", err)
	}
}

func TestInspectZipRejectsUnsafeAbsoluteAndWindowsPaths(t *testing.T) {
	tests := []string{"/etc/passwd", `C:\evil.txt`, `dir\evil.txt`}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			zipPath := filepath.Join(t.TempDir(), "unsafe.zip")
			createTestZip(t, zipPath, map[string]string{name: "nope"})
			if _, err := inspectZip(zipPath); ErrorKindOf(err) != KindValidation {
				t.Fatalf("expected validation error for unsafe path, got %v", err)
			}
		})
	}
}

func TestInspectZipEnforcesFileCountLimit(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "too-many.zip")
	entries := make(map[string]string, maxArchiveFiles+1)
	for i := 0; i < maxArchiveFiles+1; i++ {
		entries[fmt.Sprintf("files/file-%04d.txt", i)] = "x"
	}
	createTestZip(t, zipPath, entries)
	if _, err := inspectZip(zipPath); ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation error for too many files, got %v", err)
	}
}

func TestInspectZipEnforcesUncompressedSizeLimit(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "too-large.zip")
	createTestZip(t, zipPath, map[string]string{"big.txt": "x"})
	patchCentralDirectoryUncompressedSize(t, zipPath, maxArchiveUncompressedBytes+1)
	if _, err := inspectZip(zipPath); ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation error for excessive uncompressed size, got %v", err)
	}
}

func TestStoreUploadedArtifactExtractsTextMetadata(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(32*1024, 2))
	const report = "step 1\nstep 2\nresult ok"
	stored, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "report.txt",
		Reader:   bytes.NewBufferString(report),
	}, "artifact-1", "submission-1")
	if err != nil {
		t.Fatalf("storeUploadedArtifact failed: %v", err)
	}
	if stored.Kind != ArtifactKindReport {
		t.Fatalf("unexpected artifact kind: %s", stored.Kind)
	}
	if stored.SHA256Hex == "" || stored.StorageKey == "" {
		t.Fatalf("hash and storage key should be set: %+v", stored)
	}
	if stored.TextExcerpt == "" {
		t.Fatal("text excerpt should be extracted")
	}
	if !strings.HasPrefix(stored.ContentType, "text/plain") {
		t.Fatalf("expected text content type, got %q", stored.ContentType)
	}
	if err := validateContentType(".txt", stored.ContentType); err != nil {
		t.Fatalf("stored text content type should validate: %v", err)
	}
	var metadata struct {
		Parser    string `json:"parser"`
		Extension string `json:"extension"`
	}
	if err := json.Unmarshal(stored.Metadata, &metadata); err != nil {
		t.Fatalf("metadata should be valid JSON: %v", err)
	}
	if metadata.Parser != "text_excerpt" || metadata.Extension != ".txt" {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	if stored.TextExcerpt != report {
		t.Fatalf("expected full short report excerpt, got %q", stored.TextExcerpt)
	}
	longReport := strings.Repeat("A", maxTextExcerptBytes*2)
	longStored, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "long-report.txt",
		Reader:   bytes.NewBufferString(longReport),
	}, "artifact-2", "submission-1")
	if err != nil {
		t.Fatalf("storeUploadedArtifact for long text failed: %v", err)
	}
	if len(longStored.TextExcerpt) == 0 || len(longStored.TextExcerpt) >= len(longReport) || !strings.HasPrefix(longReport, longStored.TextExcerpt) {
		t.Fatalf("long text excerpt should be a non-empty truncated prefix, got length %d", len(longStored.TextExcerpt))
	}
	if _, err := store.Resolve(stored.StorageKey); err != nil {
		t.Fatalf("stored key should resolve: %v", err)
	}
}

func TestStoreUploadedArtifactExtractsImageMetadata(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(1024*1024, 2))
	imageBuffer := &bytes.Buffer{}
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(1, 1, color.RGBA{R: 0xff, A: 0xff})
	if err := png.Encode(imageBuffer, img); err != nil {
		t.Fatal(err)
	}
	stored, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "screenshot.png",
		Reader:   imageBuffer,
	}, "artifact-image", "submission-1")
	if err != nil {
		t.Fatalf("storeUploadedArtifact for image failed: %v", err)
	}
	if stored.ContentType != "image/png" {
		t.Fatalf("expected image/png content type, got %q", stored.ContentType)
	}
	var metadata struct {
		Parser    string `json:"parser"`
		Extension string `json:"extension"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	if err := json.Unmarshal(stored.Metadata, &metadata); err != nil {
		t.Fatalf("metadata should be valid JSON: %v", err)
	}
	if metadata.Parser != "image_metadata" || metadata.Extension != ".png" || metadata.Width != 3 || metadata.Height != 2 {
		t.Fatalf("unexpected image metadata: %+v", metadata)
	}
}

func TestStoreUploadedArtifactExtractsDOCXMetadataAndText(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(1024*1024, 2))
	docxBuffer := &bytes.Buffer{}
	writer := zip.NewWriter(docxBuffer)
	entry, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>实验报告</w:t></w:r></w:p><w:p><w:r><w:t>步骤一 完成部署验证</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>表格项</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`
	if _, err := entry.Write([]byte(xmlBody)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	stored, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "report.docx",
		Reader:   bytes.NewReader(docxBuffer.Bytes()),
	}, "artifact-docx", "submission-1")
	if err != nil {
		t.Fatalf("storeUploadedArtifact for docx failed: %v", err)
	}
	if stored.TextExcerpt == "" || !strings.Contains(stored.TextExcerpt, "部署验证") {
		t.Fatalf("expected docx excerpt to contain parsed text, got %q", stored.TextExcerpt)
	}
	var metadata struct {
		Parser         string `json:"parser"`
		ParagraphCount int    `json:"paragraph_count"`
		TableCount     int    `json:"table_count"`
	}
	if err := json.Unmarshal(stored.Metadata, &metadata); err != nil {
		t.Fatalf("docx metadata should be valid JSON: %v", err)
	}
	if metadata.Parser != "docx_text_excerpt" || metadata.ParagraphCount < 2 || metadata.TableCount != 1 {
		t.Fatalf("unexpected docx metadata: %+v", metadata)
	}
}

func TestStoreUploadedArtifactExtractsPDFText(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(1024*1024, 2))
	pdfBuffer := &bytes.Buffer{}
	pdfDoc := gofpdf.New("P", "mm", "A4", "")
	pdfDoc.AddPage()
	pdfDoc.SetFont("Helvetica", "", 16)
	pdfDoc.Cell(40, 10, "Deployment result ok")
	if err := pdfDoc.Output(pdfBuffer); err != nil {
		t.Fatalf("build pdf fixture: %v", err)
	}
	stored, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "report.pdf",
		Reader:   bytes.NewReader(pdfBuffer.Bytes()),
	}, "artifact-pdf", "submission-1")
	if err != nil {
		t.Fatalf("storeUploadedArtifact for pdf failed: %v", err)
	}
	if stored.TextExcerpt == "" || !strings.Contains(strings.ToLower(stored.TextExcerpt), "deployment") {
		t.Fatalf("expected pdf excerpt to contain extracted text, got %q", stored.TextExcerpt)
	}
	var metadata struct {
		Parser    string `json:"parser"`
		PageCount int    `json:"page_count"`
	}
	if err := json.Unmarshal(stored.Metadata, &metadata); err != nil {
		t.Fatalf("pdf metadata should be valid JSON: %v", err)
	}
	if metadata.Parser != "pdf_text_excerpt" || metadata.PageCount != 1 {
		t.Fatalf("unexpected pdf metadata: %+v", metadata)
	}
}

func TestStoreUploadedArtifactRejectsOversizedFile(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(4, 2))
	_, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "report.txt",
		Reader:   bytes.NewBufferString("too large"),
	}, "artifact-1", "submission-1")
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation error for oversized upload, got %v", err)
	}
}

func TestStoreUploadedArtifactRejectsUnsupportedExtensionAndKindMismatch(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(1024, 2))
	tests := []struct {
		name  string
		input ArtifactUploadInput
	}{
		{name: "unsupported", input: ArtifactUploadInput{FileName: "file.xyz", Reader: bytes.NewBufferString("content")}},
		{name: "missing extension", input: ArtifactUploadInput{FileName: "file", Reader: bytes.NewBufferString("content")}},
		{name: "kind mismatch", input: ArtifactUploadInput{FileName: "report.txt", DeclaredKind: string(ArtifactKindScreenshot), Reader: bytes.NewBufferString("content")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.storeUploadedArtifact(tc.input, "artifact-"+tc.name, "submission-1")
			if ErrorKindOf(err) != KindValidation {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestValidateGitLinkRejectsCredentials(t *testing.T) {
	_, err := validateGitLink(CreateGitLinkInput{URL: "https://user:pass@example.edu/repo.git"})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected credentials to be rejected, got %v", err)
	}
}

func TestValidateGitLinkRejectsMalformedValues(t *testing.T) {
	tests := []CreateGitLinkInput{
		{URL: "repo.git"},
		{URL: "ftp://example.edu/repo.git"},
		{URL: "https:///repo.git"},
		{URL: "https://example.edu/repo.git", CommitSHA: "abc123"},
		{URL: "https://example.edu/repo.git", CommitSHA: "zzzzzzzz"},
	}
	for _, input := range tests {
		if _, err := validateGitLink(input); ErrorKindOf(err) != KindValidation {
			t.Fatalf("expected validation error for %+v, got %v", input, err)
		}
	}
}

func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, body := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func patchCentralDirectoryUncompressedSize(t *testing.T, zipPath string, size int) {
	t.Helper()
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	signature := []byte{0x50, 0x4b, 0x01, 0x02}
	index := bytes.Index(data, signature)
	if index < 0 {
		t.Fatal("central directory header not found")
	}
	binary.LittleEndian.PutUint32(data[index+24:index+28], uint32(size))
	if err := os.WriteFile(zipPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
