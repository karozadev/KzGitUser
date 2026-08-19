// Package update checks GitHub for newer kzgit releases and can download
// and install one in place of the currently running binary.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

	// releasesPerPage caps how many releases a single API call asks
	// GitHub for. It's generous enough to cover realistic upgrade gaps
	// (a user several versions behind) without needing pagination.
	releasesPerPage = 100
)

// APIURL and ReleaseBaseURL are the GitHub endpoints used to look up and
// download releases. They are variables (rather than constants) so tests
// can point them at a local server.
var (
	// APIURL lists releases (not just the latest one), so a user several
	// versions behind gets a changelog covering every release in between,
	// not just the newest one's.
	APIURL         = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", repoOwner, repoName, releasesPerPage)
	ReleaseBaseURL = fmt.Sprintf("https://github.com/%s/%s/releases/download", repoOwner, repoName)

	// HTTPClient performs the small JSON API requests (checking releases).
	// 10s gives real-world latency more headroom than a bare API call
	// might seem to need, without letting a truly dead connection hang a
	// routine command for long.
	HTTPClient = &http.Client{Timeout: 10 * time.Second}

	// DownloadClient performs the larger release-archive downloads, which
	// need a much longer timeout than the API check — a multi-megabyte
	// binary can easily take longer than 10s on a slow connection.
	DownloadClient = &http.Client{Timeout: 2 * time.Minute}
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
// is missing, stale, or was computed for a different current version,
// keeping routine calls (e.g. from `kzgit whoami`) cheap and
// network-failure-tolerant.
func Check(current string) (Info, error) {
	release, ok := cachedRelease(current)
	if !ok {
		fetched, err := fetchRelease(current)
		if err != nil {
			return Info{}, err
		}
		release = fetched
		_ = writeCache(current, release)
	}
	return newInfo(current, release), nil
}

// ForceCheck always queries GitHub for the latest release, bypassing the
// cache, and refreshes the cache with the result.
func ForceCheck(current string) (Info, error) {
	release, err := fetchRelease(current)
	if err != nil {
		return Info{}, err
	}
	_ = writeCache(current, release)
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

// releaseInfo is a released version paired with its (possibly cumulative)
// changelog.
type releaseInfo struct {
	Version string
	Notes   string
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// fetchRelease lists every published, non-prerelease release and returns
// the latest one's version alongside a changelog that concatenates every
// release newer than current — not just the single newest release — so
// someone upgrading across several versions sees everything that changed
// along the way, oldest to newest.
func fetchRelease(current string) (releaseInfo, error) {
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

	var all []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return releaseInfo{}, fmt.Errorf("parsing release info: %w", err)
	}

	return aggregateReleases(all, current)
}

func aggregateReleases(all []githubRelease, current string) (releaseInfo, error) {
	type versioned struct {
		version string
		body    string
	}

	var latest string
	var newerThanCurrent []versioned

	for _, r := range all {
		if r.Draft || r.Prerelease {
			continue
		}
		v := trimV(r.TagName)
		tag := "v" + v
		if !semver.IsValid(tag) {
			continue
		}
		if latest == "" || semver.Compare(tag, "v"+latest) > 0 {
			latest = v
		}
		if body := strings.TrimSpace(r.Body); isNewer(current, v) && body != "" {
			newerThanCurrent = append(newerThanCurrent, versioned{version: v, body: body})
		}
	}

	if latest == "" {
		return releaseInfo{}, fmt.Errorf("no published releases found")
	}

	sort.Slice(newerThanCurrent, func(i, j int) bool {
		return semver.Compare("v"+newerThanCurrent[i].version, "v"+newerThanCurrent[j].version) < 0
	})

	var notes strings.Builder
	for i, e := range newerThanCurrent {
		if i > 0 {
			notes.WriteString("\n\n")
		}
		fmt.Fprintf(&notes, "## v%s\n%s", e.version, e.body)
	}

	return releaseInfo{Version: latest, Notes: notes.String()}, nil
}

type cacheData struct {
	CheckedAt time.Time `json:"checkedAt"`
	Current   string    `json:"current"`
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

// cachedRelease returns the cached release if it's fresh (within
// checkInterval) and was computed for the same current version — a cached
// changelog computed relative to a different version would be wrong, since
// which releases count as "newer than current" depends on current itself.
func cachedRelease(current string) (releaseInfo, bool) {
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
	if time.Since(c.CheckedAt) > checkInterval || c.Latest == "" || c.Current != current {
		return releaseInfo{}, false
	}
	return releaseInfo{Version: c.Latest, Notes: c.Notes}, true
}

func writeCache(current string, release releaseInfo) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cacheData{CheckedAt: time.Now(), Current: current, Latest: release.Version, Notes: release.Notes})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
