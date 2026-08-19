package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func buildTarGz(t *testing.T, binaryName string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{Name: binaryName, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func buildZip(t *testing.T, binaryName string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	fw, err := zw.Create(binaryName)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// newInstallServer serves a release archive (matching the current
// runtime.GOOS/GOARCH so archiveName() resolves to it) and its
// checksums.txt at the paths Install() expects.
func newInstallServer(t *testing.T, version string, binaryContent []byte) *httptest.Server {
	t.Helper()

	assetName, err := archiveName(version)
	if err != nil {
		t.Fatalf("archiveName: %v", err)
	}

	binaryFile := binaryName
	if runtime.GOOS == "windows" {
		binaryFile += ".exe"
	}

	var archiveBytes []byte
	if runtime.GOOS == "windows" {
		archiveBytes = buildZip(t, binaryFile, binaryContent)
	} else {
		archiveBytes = buildTarGz(t, binaryFile, binaryContent)
	}

	checksums := fmt.Sprintf("%s  %s\n", sha256Hex(archiveBytes), assetName)

	mux := http.NewServeMux()
	tag := "v" + version
	mux.HandleFunc("/"+tag+"/"+assetName, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archiveBytes)
	})
	mux.HandleFunc("/"+tag+"/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksums))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestInstall_FullPipeline(t *testing.T) {
	isolatedEnv(t)

	content := []byte("fake released binary")
	srv := newInstallServer(t, "1.2.3", content)
	ReleaseBaseURL = srv.URL

	var capturedPath string
	origReplace := replaceExecutableFunc
	replaceExecutableFunc = func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(data) != string(content) {
			t.Fatalf("unexpected extracted binary content: %q", data)
		}
		capturedPath = path
		return nil
	}
	t.Cleanup(func() { replaceExecutableFunc = origReplace })

	if err := Install("1.2.3"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if capturedPath == "" {
		t.Fatal("expected replaceExecutableFunc to be called with the extracted binary path")
	}
}

func TestInstall_ChecksumsDownloadFails(t *testing.T) {
	isolatedEnv(t)

	assetName, err := archiveName("1.2.3")
	if err != nil {
		t.Fatalf("archiveName: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1.2.3/"+assetName, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buildTarGz(t, binaryName, []byte("content")))
	})
	// No handler for checksums.txt: the mux's default 404 kicks in.
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ReleaseBaseURL = srv.URL

	if err := Install("1.2.3"); err == nil {
		t.Fatal("expected Install to fail when checksums.txt can't be downloaded")
	}
}

func TestInstall_ChecksumMismatchAborts(t *testing.T) {
	isolatedEnv(t)

	assetName, err := archiveName("1.2.3")
	if err != nil {
		t.Fatalf("archiveName: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1.2.3/"+assetName, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buildTarGz(t, binaryName, []byte("content")))
	})
	mux.HandleFunc("/v1.2.3/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0000000000000000000000000000000000000000000000000000000000000000  " + assetName + "\n"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ReleaseBaseURL = srv.URL

	called := false
	origReplace := replaceExecutableFunc
	replaceExecutableFunc = func(path string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { replaceExecutableFunc = origReplace })

	if err := Install("1.2.3"); err == nil {
		t.Fatal("expected Install to fail on checksum mismatch")
	}
	if called {
		t.Fatal("expected replaceExecutableFunc not to be called when the checksum doesn't match")
	}
}

func TestInstall_ReplacesExecutable(t *testing.T) {
	isolatedEnv(t)

	// Simulate the "currently running executable" with a fake binary on
	// disk, and point os.Executable()'s result at it indirectly by
	// exercising replaceExecutable through a temp HOME-relative path isn't
	// possible (os.Executable is not overridable), so we test the lower
	// layer (replaceExecutable) directly using a fake current path.
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "kzgit-current")
	if err := os.WriteFile(currentPath, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("seed current binary: %v", err)
	}

	newContent := []byte("new binary content")
	newPath := filepath.Join(dir, "kzgit-new")
	if err := os.WriteFile(newPath, newContent, 0o644); err != nil {
		t.Fatalf("seed new binary: %v", err)
	}

	if err := copyReplace(newPath, currentPath); err != nil {
		t.Fatalf("copyReplace: %v", err)
	}

	got, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("reading replaced binary: %v", err)
	}
	if string(got) != string(newContent) {
		t.Fatalf("current binary was not replaced: got %q", got)
	}

	info, err := os.Stat(currentPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("expected replaced binary to be executable, got mode %v", info.Mode())
	}
}

func TestVerifyChecksum_Success(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello world")
	archivePath := filepath.Join(dir, "archive.tar.gz")
	if err := os.WriteFile(archivePath, content, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	checksumsPath := filepath.Join(dir, "checksums.txt")
	line := fmt.Sprintf("%s  archive.tar.gz\n", sha256Hex(content))
	if err := os.WriteFile(checksumsPath, []byte(line), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	if err := verifyChecksum(archivePath, checksumsPath, "archive.tar.gz"); err != nil {
		t.Fatalf("verifyChecksum: %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.tar.gz")
	if err := os.WriteFile(archivePath, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	checksumsPath := filepath.Join(dir, "checksums.txt")
	line := "0000000000000000000000000000000000000000000000000000000000000000  archive.tar.gz\n"
	if err := os.WriteFile(checksumsPath, []byte(line), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	err := verifyChecksum(archivePath, checksumsPath, "archive.tar.gz")
	if err == nil {
		t.Fatal("expected a checksum mismatch error")
	}
}

func TestVerifyChecksum_MissingEntry(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.tar.gz")
	if err := os.WriteFile(archivePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	checksumsPath := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte("abc  other-file.tar.gz\n"), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	if err := verifyChecksum(archivePath, checksumsPath, "archive.tar.gz"); err == nil {
		t.Fatal("expected an error when the checksum entry is missing")
	}
}

func TestExtractBinary_TarGz(t *testing.T) {
	dir := t.TempDir()
	content := []byte("binary-content")
	archiveBytes := buildTarGz(t, binaryName, content)
	archivePath := filepath.Join(dir, "kzgit_0.1.0_linux_amd64.tar.gz")
	if err := os.WriteFile(archivePath, archiveBytes, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	outPath, err := extractBinary(archivePath, dir)
	if err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected extracted content: %q", got)
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	dir := t.TempDir()
	content := []byte("binary-content")
	archiveBytes := buildZip(t, binaryName+".exe", content)
	archivePath := filepath.Join(dir, "kzgit_0.1.0_windows_amd64.zip")
	if err := os.WriteFile(archivePath, archiveBytes, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	// extractBinary picks the wanted filename based on runtime.GOOS, so on
	// non-Windows test runners it looks for "kzgit" not "kzgit.exe". Test
	// the zip extraction path directly instead.
	outPath := filepath.Join(dir, "out")
	if err := extractFromZip(archivePath, binaryName+".exe", outPath); err != nil {
		t.Fatalf("extractFromZip: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected extracted content: %q", got)
	}
}

func TestArchiveNameFor_UnsupportedPlatform(t *testing.T) {
	if _, err := archiveNameFor("1.0.0", "plan9", "amd64"); err == nil {
		t.Fatal("expected an error for an unsupported platform")
	}
}

func TestExtractBinaryFor_Windows(t *testing.T) {
	dir := t.TempDir()
	content := []byte("windows binary")
	archiveBytes := buildZip(t, "kzgit.exe", content)
	archivePath := filepath.Join(dir, "kzgit_0.1.0_windows_amd64.zip")
	if err := os.WriteFile(archivePath, archiveBytes, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	outPath, err := extractBinaryFor(archivePath, dir, "windows")
	if err != nil {
		t.Fatalf("extractBinaryFor: %v", err)
	}
	if filepath.Base(outPath) != "kzgit.exe" {
		t.Fatalf("expected kzgit.exe, got %s", outPath)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestExtractFromTarGz_NotFound(t *testing.T) {
	dir := t.TempDir()
	archiveBytes := buildTarGz(t, "some-other-file", []byte("x"))
	archivePath := filepath.Join(dir, "archive.tar.gz")
	if err := os.WriteFile(archivePath, archiveBytes, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	if err := extractFromTarGz(archivePath, binaryName, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected an error when the wanted file isn't in the tar.gz archive")
	}
}

func TestExtractFromZip_NotFound(t *testing.T) {
	dir := t.TempDir()
	archiveBytes := buildZip(t, "some-other-file", []byte("x"))
	archivePath := filepath.Join(dir, "archive.zip")
	if err := os.WriteFile(archivePath, archiveBytes, 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	if err := extractFromZip(archivePath, binaryName, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected an error when the wanted file isn't in the zip archive")
	}
}

func TestWriteExtracted_OpenFileError(t *testing.T) {
	err := writeExtracted(filepath.Join(t.TempDir(), "nonexistent-dir", "out"), bytes.NewReader([]byte("x")))
	if err == nil {
		t.Fatal("expected an error when the output path's directory doesn't exist")
	}
}

func TestCopyReplace_StagedFileError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	dest := filepath.Join(dir, "nonexistent-dir", "dest")
	if err := copyReplace(src, dest); err == nil {
		t.Fatal("expected an error when the staged file's directory doesn't exist")
	}
}

func TestArchiveName(t *testing.T) {
	name, err := archiveName("1.2.3")
	if err != nil {
		t.Fatalf("archiveName: %v", err)
	}
	switch runtime.GOOS {
	case "windows":
		if name != "kzgit_1.2.3_windows_"+runtime.GOARCH+".zip" {
			t.Fatalf("unexpected archive name: %s", name)
		}
	default:
		want := fmt.Sprintf("kzgit_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
		if name != want {
			t.Fatalf("unexpected archive name: %s, want %s", name, want)
		}
	}
}

func TestDownloadFile(t *testing.T) {
	isolatedEnv(t)
	content := []byte("downloaded content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := downloadFile(srv.URL, dest); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestDownloadFile_UsesDownloadClientNotAPIClient(t *testing.T) {
	isolatedEnv(t)
	content := []byte("a slow but real download")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write(content)
	}))
	t.Cleanup(srv.Close)

	// HTTPClient (the small API-check client) has a timeout far shorter
	// than the server's delay; DownloadClient does not. If downloadFile
	// used HTTPClient, this would time out and fail.
	HTTPClient = &http.Client{Timeout: 1 * time.Millisecond}

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := downloadFile(srv.URL, dest); err != nil {
		t.Fatalf("downloadFile: %v (should use DownloadClient's longer timeout)", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestDownloadFile_NotFound(t *testing.T) {
	isolatedEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := downloadFile(srv.URL, dest); err == nil {
		t.Fatal("expected an error for a 404 response")
	}
}
