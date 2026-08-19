package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
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
