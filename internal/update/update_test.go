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
// and points APIURL/ReleaseBaseURL/HTTPClient at test doubles, restoring
// the real values on cleanup.
func isolatedEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	origAPI, origBase, origClient := APIURL, ReleaseBaseURL, HTTPClient
	t.Cleanup(func() {
		APIURL, ReleaseBaseURL, HTTPClient = origAPI, origBase, origClient
	})
}

func newReleaseServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releaseResponse{TagName: tag})
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

func TestCheck_UsesCacheWithinInterval(t *testing.T) {
	isolatedEnv(t)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releaseResponse{TagName: "v0.2.0"})
	}))
	t.Cleanup(srv.Close)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releaseResponse{TagName: "v0.2.0"})
	}))
	t.Cleanup(srv.Close)
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

	if err := writeCache("0.1.0"); err == nil {
		t.Fatal("expected an error when the cache directory can't be created")
	}
}

func TestCachedLatest_CorruptedCache(t *testing.T) {
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

	latest, err := cachedLatest()
	if err != nil {
		t.Fatalf("cachedLatest: %v", err)
	}
	if latest != "" {
		t.Fatalf("expected empty result for a corrupted cache, got %q", latest)
	}
}

func TestCachedLatest_Stale(t *testing.T) {
	isolatedEnv(t)
	if err := writeCache("0.5.0"); err != nil {
		t.Fatalf("writeCache: %v", err)
	}

	// Backdate the cache file well past checkInterval.
	path, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	data, err := json.Marshal(cacheData{CheckedAt: time.Now().Add(-48 * time.Hour), Latest: "0.5.0"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write stale cache: %v", err)
	}

	latest, err := cachedLatest()
	if err != nil {
		t.Fatalf("cachedLatest: %v", err)
	}
	if latest != "" {
		t.Fatalf("expected a stale cache to be ignored, got %q", latest)
	}
}

func TestFetchLatest_Unreachable(t *testing.T) {
	isolatedEnv(t)
	APIURL = "http://127.0.0.1:1/unreachable"
	origClient := HTTPClient
	HTTPClient = &http.Client{Timeout: 500 * time.Millisecond}
	t.Cleanup(func() { HTTPClient = origClient })

	if _, err := fetchLatest(); err == nil {
		t.Fatal("expected an error when the release API is unreachable")
	}
}

func TestFetchLatest_MalformedJSON(t *testing.T) {
	isolatedEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	APIURL = srv.URL

	if _, err := fetchLatest(); err == nil {
		t.Fatal("expected an error for a malformed JSON response")
	}
}

func TestFetchLatest_EmptyTagName(t *testing.T) {
	isolatedEnv(t)
	srv := newReleaseServer(t, "")
	APIURL = srv.URL

	if _, err := fetchLatest(); err == nil {
		t.Fatal("expected an error for an empty tag_name")
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
