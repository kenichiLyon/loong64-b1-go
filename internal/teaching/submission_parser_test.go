package teaching

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kenichiLyon/loong64-b1-go/internal/storage"
)

func TestInspectZipRejectsPathTraversal(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	createTestZip(t, zipPath, map[string]string{"../evil.txt": "nope"})
	if _, err := inspectZip(zipPath); ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation error for zip slip entry, got %v", err)
	}
}

func TestStoreUploadedArtifactExtractsTextMetadata(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(1024, 2))
	stored, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "report.txt",
		Reader:   bytes.NewBufferString("step 1\nstep 2\nresult ok"),
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
	if _, err := store.Resolve(stored.StorageKey); err != nil {
		t.Fatalf("stored key should resolve: %v", err)
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

func TestValidateGitLinkRejectsCredentials(t *testing.T) {
	_, err := validateGitLink(CreateGitLinkInput{URL: "https://user:pass@example.edu/repo.git"})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected credentials to be rejected, got %v", err)
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
