// Package update checks GitHub for newer kzgit releases and can download
// and install one in place of the currently running binary.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	Current      string
	Latest       string
	Available    bool
	ReleaseNotes string
}

// Check reports whether a newer kzgit release than current is available.
// It consults an on-disk cache first and only queries GitHub if the cache
// is missing or older than checkInterval, keeping routine calls (e.g. from
// `kzgit whoami`) cheap and network-failure-tolerant.
func Check(current string) (Info, error) {
	release, ok := cachedRelease()
	if !ok {
		fetched, err := fetchLatest()
		if err != nil {
			return Info{}, err
		}
		release = fetched
		_ = writeCache(release)
	}
	return newInfo(current, release), nil
}

// ForceCheck always queries GitHub for the latest release, bypassing the
// cache, and refreshes the cache with the result.
func ForceCheck(current string) (Info, error) {
	release, err := fetchLatest()
	if err != nil {
		return Info{}, err
	}
	_ = writeCache(release)
	return newInfo(current, release), nil
}

func newInfo(current string, release releaseInfo) Info {
	return Info{
		Current:      current,
		Latest:       release.Version,
		Available:    isNewer(current, release.Version),
		ReleaseNotes: strings.TrimSpace(release.Notes),
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

// releaseInfo is a released version paired with its GitHub release notes.
type releaseInfo struct {
	Version string
	Notes   string
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

func fetchLatest() (releaseInfo, error) {
	req, err := http.NewRequest(http.MethodGet, APIURL, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return releaseInfo{}, fmt.Errorf("checking latest kzgit release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return releaseInfo{}, fmt.Errorf("checking latest kzgit release: unexpected status %s", resp.Status)
	}

	var rel releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return releaseInfo{}, fmt.Errorf("parsing release info: %w", err)
	}
	if rel.TagName == "" {
		return releaseInfo{}, fmt.Errorf("no release tag found")
	}
	return releaseInfo{Version: trimV(rel.TagName), Notes: rel.Body}, nil
}

type cacheData struct {
	CheckedAt time.Time `json:"checkedAt"`
	Latest    string    `json:"latest"`
	Notes     string    `json:"notes,omitempty"`
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

func cachedRelease() (releaseInfo, bool) {
	path, err := cachePath()
	if err != nil {
		return releaseInfo{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return releaseInfo{}, false
	}
	var c cacheData
	if err := json.Unmarshal(data, &c); err != nil {
		return releaseInfo{}, false
	}
	if time.Since(c.CheckedAt) > checkInterval || c.Latest == "" {
		return releaseInfo{}, false
	}
	return releaseInfo{Version: c.Latest, Notes: c.Notes}, true
}

func writeCache(release releaseInfo) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cacheData{CheckedAt: time.Now(), Latest: release.Version, Notes: release.Notes})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
