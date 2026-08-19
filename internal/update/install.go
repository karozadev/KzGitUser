package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const binaryName = "kzgit"

// Install downloads the given released version (without a leading "v"),
// verifies its checksum against the release's published checksums.txt, and
// atomically replaces the currently running kzgit executable with it.
func Install(version string) error {
	assetName, err := archiveName(version)
	if err != nil {
		return err
	}
	tag := "v" + version

	tmpDir, err := os.MkdirTemp("", "kzgit-update-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(fmt.Sprintf("%s/%s/%s", ReleaseBaseURL, tag, assetName), archivePath); err != nil {
		return fmt.Errorf("downloading %s: %w", assetName, err)
	}

	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(fmt.Sprintf("%s/%s/checksums.txt", ReleaseBaseURL, tag), checksumsPath); err != nil {
		return fmt.Errorf("downloading checksums.txt: %w", err)
	}

	if err := verifyChecksum(archivePath, checksumsPath, assetName); err != nil {
		return err
	}

	binaryPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		return err
	}

	return replaceExecutableFunc(binaryPath)
}

// replaceExecutableFunc defaults to replaceExecutable; tests substitute a
// stub so Install's download/verify/extract pipeline can be exercised
// end-to-end without ever touching the real running executable.
var replaceExecutableFunc = replaceExecutable

func archiveName(version string) (string, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	switch goos {
	case "linux", "darwin":
		return fmt.Sprintf("%s_%s_%s_%s.tar.gz", binaryName, version, goos, arch), nil
	case "windows":
		return fmt.Sprintf("%s_%s_%s_%s.zip", binaryName, version, goos, arch), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", goos, arch)
	}
}

func downloadFile(url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

func verifyChecksum(archivePath, checksumsPath, assetName string) error {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return err
	}

	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum for %s not found in checksums.txt", assetName)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))

	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}
	return nil
}

func extractBinary(archivePath, destDir string) (string, error) {
	binaryFile := binaryName
	if runtime.GOOS == "windows" {
		binaryFile += ".exe"
	}
	outPath := filepath.Join(destDir, binaryFile)

	var err error
	if strings.HasSuffix(archivePath, ".zip") {
		err = extractFromZip(archivePath, binaryFile, outPath)
	} else {
		err = extractFromTarGz(archivePath, binaryFile, outPath)
	}
	if err != nil {
		return "", err
	}
	return outPath, nil
}

func extractFromTarGz(archivePath, wantName, outPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s not found in archive", wantName)
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) != wantName {
			continue
		}
		return writeExtracted(outPath, tr)
	}
}

func extractFromZip(archivePath, wantName, outPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		if filepath.Base(f.Name) != wantName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() { _ = rc.Close() }()
		return writeExtracted(outPath, rc)
	}
	return fmt.Errorf("%s not found in archive", wantName)
}

func writeExtracted(outPath string, r io.Reader) error {
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, r)
	return err
}

// replaceExecutable atomically swaps the currently running kzgit binary for
// newBinaryPath. The replacement is written into the same directory as the
// current executable (via copyReplace) so the final rename is guaranteed to
// be on the same filesystem, and therefore atomic, even though the
// downloaded file itself lives in a temp directory that may be on a
// different filesystem.
func replaceExecutable(newBinaryPath string) error {
	currentPath, err := os.Executable()
	if err != nil {
		return err
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return err
	}

	if err := os.Chmod(newBinaryPath, 0o755); err != nil {
		return err
	}

	return copyReplace(newBinaryPath, currentPath)
}

func copyReplace(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	staged := dest + ".new"
	out, err := os.OpenFile(staged, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("writing %s (you may need administrator/sudo rights): %w", staged, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(staged)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	if err := os.Rename(staged, dest); err != nil {
		_ = os.Remove(staged)
		return fmt.Errorf("replacing %s (it may still be in use, or you may need administrator/sudo rights): %w", dest, err)
	}
	return nil
}
