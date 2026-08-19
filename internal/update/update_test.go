package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// isolatedEnv redirects the update-check cache to a fresh temp directory
// and points APIURL/ReleaseBaseURL/HTTPClient/DownloadClient at test
// doubles, restoring the real values on cleanup.
func isolatedEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	origAPI, origBase, origClient, origDownload := APIURL, ReleaseBaseURL, HTTPClient, DownloadClient
	t.Cleanup(func() {
		APIURL, ReleaseBaseURL, HTTPClient, DownloadClient = origAPI, origBase, origClient, origDownload
	})
}

// newReleaseServer serves a single release (as GitHub's /releases list
// endpoint would, as a one-element JSON array) at the given tag with no
// notes.
func newReleaseServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	return newReleasesServer(t, githubRelease{TagName: tag})
}

// newReleaseServerWithNotes serves a single release with a body.
func newReleaseServerWithNotes(t *testing.T, tag, notes string) *httptest.Server {
	t.Helper()
	return newReleasesServer(t, githubRelease{TagName: tag, Body: notes})
}

// newReleasesServer serves the given releases as GitHub's /releases list
// endpoint would: a JSON array, in the order given.
func newReleasesServer(t *testing.T, releases ...githubRelease) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releases)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCheck_NewerAvailable(t *testing.T) {
	isolatedEnv(t)
	srv := newReleaseServer(t, "v0.2.0")
	APIURL = srv.URL

	info, err := Check("0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Available {
		t.Fatalf("expected an update to be available, got %+v", info)
	}
	if info.Latest != "0.2.0" {
		t.Fatalf("expected latest=0.2.0, got %q", info.Latest)
	}
}

func TestCheck_CarriesReleaseNotes(t *testing.T) {
	isolatedEnv(t)
	srv := newReleaseServerWithNotes(t, "v0.2.0", "  ## What's new\n- feat: self-update\n  ")
	APIURL = srv.URL

	info, err := Check("0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.ReleaseNotes != "## v0.2.0\n## What's new\n- feat: self-update" {
		t.Fatalf("unexpected release notes: %q", info.ReleaseNotes)
	}
}

func TestCheck_ReleaseNotesSurviveCache(t *testing.T) {
	isolatedEnv(t)
	srv := newReleaseServerWithNotes(t, "v0.2.0", "release notes here")
	APIURL = srv.URL

	if _, err := Check("0.1.0"); err != nil {
		t.Fatalf("first Check: %v", err)
	}
	// Second call is served from cache; notes should still be present.
	info, err := Check("0.1.0")
	if err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if info.ReleaseNotes != "## v0.2.0\nrelease notes here" {
		t.Fatalf("expected cached release notes, got %q", info.ReleaseNotes)
	}
}

func TestCheck_CacheInvalidatedByDifferentCurrent(t *testing.T) {
	isolatedEnv(t)
	calls := 0
	srv := newCountingReleasesServer(t, &calls, githubRelease{TagName: "v0.3.0"})
	APIURL = srv.URL

	if _, err := Check("0.1.0"); err != nil {
		t.Fatalf("first Check: %v", err)
	}
	// A different "current" invalidates the cache, since which releases
	// count as "newer" (and thus the changelog) depends on it.
	if _, err := Check("0.2.0"); err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected a fresh fetch when current changes, got %d calls", calls)
	}
}

func TestAggregateReleases_CumulativeAcrossVersions(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v0.3.0", Body: "changelog for 0.3.0"},
		{TagName: "v0.2.0", Body: "changelog for 0.2.0"},
		{TagName: "v0.1.0", Body: "changelog for 0.1.0"},
	}

	release, err := aggregateReleases(releases, "0.1.0")
	if err != nil {
		t.Fatalf("aggregateReleases: %v", err)
	}
	if release.Version != "0.3.0" {
		t.Fatalf("expected latest=0.3.0, got %q", release.Version)
	}

	want := "## v0.2.0\nchangelog for 0.2.0\n\n## v0.3.0\nchangelog for 0.3.0"
	if release.Notes != want {
		t.Fatalf("expected cumulative, oldest-first changelog:\nwant: %q\ngot:  %q", want, release.Notes)
	}
}

func TestAggregateReleases_SkipsDraftsAndPrereleases(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v0.3.0", Body: "stable"},
		{TagName: "v0.4.0-beta.1", Body: "beta", Prerelease: true},
		{TagName: "v0.5.0", Body: "unpublished draft", Draft: true},
	}

	release, err := aggregateReleases(releases, "0.1.0")
	if err != nil {
		t.Fatalf("aggregateReleases: %v", err)
	}
	if release.Version != "0.3.0" {
		t.Fatalf("expected drafts/prereleases to be skipped, latest should be 0.3.0, got %q", release.Version)
	}
}

func TestAggregateReleases_NoPublishedReleases(t *testing.T) {
	_, err := aggregateReleases(nil, "0.1.0")
	if err == nil {
		t.Fatal("expected an error when there are no published releases")
	}
}

func TestAggregateReleases_IgnoresMalformedTags(t *testing.T) {
	releases := []githubRelease{
		{TagName: "not-a-version"},
		{TagName: "v0.2.0", Body: "ok release"},
	}
	release, err := aggregateReleases(releases, "0.1.0")
	if err != nil {
		t.Fatalf("aggregateReleases: %v", err)
	}
	if release.Version != "0.2.0" {
		t.Fatalf("expected malformed tags to be ignored, got %q", release.Version)
	}
}

func TestCheck_UpToDate(t *testing.T) {
	isolatedEnv(t)
	srv := newReleaseServer(t, "v0.1.0")
	APIURL = srv.URL

	info, err := Check("0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Available {
		t.Fatalf("expected no update to be available, got %+v", info)
	}
}

func TestCheck_DevAlwaysOutdated(t *testing.T) {
	isolatedEnv(t)
	srv := newReleaseServer(t, "v0.1.0")
	APIURL = srv.URL

	info, err := Check("dev")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Available {
		t.Fatal("expected a 'dev' build to always be considered outdated")
	}
}

func newCountingReleasesServer(t *testing.T, calls *int, releases ...githubRelease) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releases)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCheck_UsesCacheWithinInterval(t *testing.T) {
	isolatedEnv(t)
	calls := 0
	srv := newCountingReleasesServer(t, &calls, githubRelease{TagName: "v0.2.0"})
	APIURL = srv.URL

	if _, err := Check("0.1.0"); err != nil {
		t.Fatalf("first Check: %v", err)
	}
	if _, err := Check("0.1.0"); err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 network call due to caching, got %d", calls)
	}
}

func TestForceCheck_BypassesCache(t *testing.T) {
	isolatedEnv(t)
	calls := 0
	srv := newCountingReleasesServer(t, &calls, githubRelease{TagName: "v0.2.0"})
	APIURL = srv.URL

	if _, err := Check("0.1.0"); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := ForceCheck("0.1.0"); err != nil {
		t.Fatalf("ForceCheck: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected ForceCheck to bypass the cache and make its own call, got %d calls", calls)
	}
}

func TestCheck_NetworkError(t *testing.T) {
	isolatedEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	APIURL = srv.URL

	if _, err := Check("0.1.0"); err == nil {
		t.Fatal("expected an error when the release API is unreachable/erroring")
	}
}

func TestCachePath_NoHomeDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	_ = os.Unsetenv("HOME")

	if _, err := cachePath(); err == nil {
		t.Fatal("expected an error when no home directory can be determined")
	}
}

func TestWriteCache_MkdirAllError(t *testing.T) {
	isolatedEnv(t)
	dir := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(dir, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := writeCache("0.1.0", releaseInfo{Version: "0.1.0"}); err == nil {
		t.Fatal("expected an error when the cache directory can't be created")
	}
}

func TestCachedRelease_CorruptedCache(t *testing.T) {
	isolatedEnv(t)
	path, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write corrupted cache: %v", err)
	}

	_, ok := cachedRelease("0.1.0")
	if ok {
		t.Fatal("expected no cached release for a corrupted cache file")
	}
}

func TestCachedRelease_Stale(t *testing.T) {
	isolatedEnv(t)
	if err := writeCache("0.1.0", releaseInfo{Version: "0.5.0"}); err != nil {
		t.Fatalf("writeCache: %v", err)
	}

	// Backdate the cache file well past checkInterval.
	path, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	data, err := json.Marshal(cacheData{CheckedAt: time.Now().Add(-48 * time.Hour), Current: "0.1.0", Latest: "0.5.0"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write stale cache: %v", err)
	}

	_, ok := cachedRelease("0.1.0")
	if ok {
		t.Fatal("expected a stale cache to be ignored")
	}
}

func TestCachedRelease_DifferentCurrent(t *testing.T) {
	isolatedEnv(t)
	if err := writeCache("0.1.0", releaseInfo{Version: "0.5.0"}); err != nil {
		t.Fatalf("writeCache: %v", err)
	}

	if _, ok := cachedRelease("0.2.0"); ok {
		t.Fatal("expected a cache entry computed for a different 'current' to be rejected")
	}
	if _, ok := cachedRelease("0.1.0"); !ok {
		t.Fatal("expected the cache entry to be valid for the version it was computed for")
	}
}

func TestFetchRelease_Unreachable(t *testing.T) {
	isolatedEnv(t)
	APIURL = "http://127.0.0.1:1/unreachable"
	origClient := HTTPClient
	HTTPClient = &http.Client{Timeout: 500 * time.Millisecond}
	t.Cleanup(func() { HTTPClient = origClient })

	if _, err := fetchRelease("0.1.0"); err == nil {
		t.Fatal("expected an error when the release API is unreachable")
	}
}

func TestFetchRelease_MalformedJSON(t *testing.T) {
	isolatedEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	APIURL = srv.URL

	if _, err := fetchRelease("0.1.0"); err == nil {
		t.Fatal("expected an error for a malformed JSON response")
	}
}

func TestFetchRelease_NoReleases(t *testing.T) {
	isolatedEnv(t)
	srv := newReleasesServer(t)
	APIURL = srv.URL

	if _, err := fetchRelease("0.1.0"); err == nil {
		t.Fatal("expected an error when there are no releases at all")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.1.0", "0.2.0", true},
		{"0.2.0", "0.1.0", false},
		{"1.0.0", "1.0.0", false},
		{"1.0.0-beta.1", "1.0.0", true},
		{"dev", "0.0.1", true},
		{"", "0.0.1", true},
	}
	for _, c := range cases {
		if got := isNewer(c.current, c.latest); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}
