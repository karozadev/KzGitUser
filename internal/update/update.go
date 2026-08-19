// Package update checks GitHub for newer kzgit releases and can download
// and install one in place of the currently running binary.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/mod/semver"
)

const (
	repoOwner = "karozadev"
	repoName  = "KzGitUser"

	// checkInterval bounds how often Check will hit the GitHub API; within
	// this window it serves the last known answer from an on-disk cache.
	checkInterval = 24 * time.Hour
)

// APIURL and ReleaseBaseURL are the GitHub endpoints used to look up and
// download releases. They are variables (rather than constants) so tests
// can point them at a local server.
var (
	APIURL         = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	ReleaseBaseURL = fmt.Sprintf("https://github.com/%s/%s/releases/download", repoOwner, repoName)

	// HTTPClient performs all network requests made by this package.
	HTTPClient = &http.Client{Timeout: 5 * time.Second}
)

// Info describes the outcome of an update check.
type Info struct {
	Current   string
	Latest    string
	Available bool
}

// Check reports whether a newer kzgit release than current is available.
// It consults an on-disk cache first and only queries GitHub if the cache
// is missing or older than checkInterval, keeping routine calls (e.g. from
// `kzgit whoami`) cheap and network-failure-tolerant.
func Check(current string) (Info, error) {
	latest, err := cachedLatest()
	if err != nil || latest == "" {
		latest, err = fetchLatest()
		if err != nil {
			return Info{}, err
		}
		_ = writeCache(latest)
	}
	return newInfo(current, latest), nil
}

// ForceCheck always queries GitHub for the latest release, bypassing the
// cache, and refreshes the cache with the result.
func ForceCheck(current string) (Info, error) {
	latest, err := fetchLatest()
	if err != nil {
		return Info{}, err
	}
	_ = writeCache(latest)
	return newInfo(current, latest), nil
}

func newInfo(current, latest string) Info {
	return Info{
		Current:   current,
		Latest:    latest,
		Available: isNewer(current, latest),
	}
}

func isNewer(current, latest string) bool {
	if current == "" || current == "dev" {
		return true
	}
	c, l := "v"+trimV(current), "v"+trimV(latest)
	if !semver.IsValid(c) || !semver.IsValid(l) {
		return current != latest
	}
	return semver.Compare(l, c) > 0
}

func trimV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

func fetchLatest() (string, error) {
	req, err := http.NewRequest(http.MethodGet, APIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("checking latest kzgit release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checking latest kzgit release: unexpected status %s", resp.Status)
	}

	var rel releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("parsing release info: %w", err)
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no release tag found")
	}
	return trimV(rel.TagName), nil
}

type cacheData struct {
	CheckedAt time.Time `json:"checkedAt"`
	Latest    string    `json:"latest"`
}

func cachePath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "kzgit", "update-check.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kzgit", "update-check.json"), nil
}

func cachedLatest() (string, error) {
	path, err := cachePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	var c cacheData
	if err := json.Unmarshal(data, &c); err != nil {
		return "", nil
	}
	if time.Since(c.CheckedAt) > checkInterval {
		return "", nil
	}
	return c.Latest, nil
}

func writeCache(latest string) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cacheData{CheckedAt: time.Now(), Latest: latest})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
