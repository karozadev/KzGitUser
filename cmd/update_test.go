package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kzupdate "github.com/karoza/kz-git-user/internal/update"
)

// isolatedUpdateEnv points internal/update's API endpoint at a local server
// returning tag, restoring the real endpoint on cleanup. It relies on
// isolatedEnv (in testutil_test.go) already having redirected the cache
// location via XDG_CONFIG_HOME.
func isolatedUpdateEnv(t *testing.T, tag string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": tag})
	}))
	t.Cleanup(srv.Close)

	origAPI := kzupdate.APIURL
	kzupdate.APIURL = srv.URL
	t.Cleanup(func() { kzupdate.APIURL = origAPI })
}

func TestUpdate_AlreadyLatest(t *testing.T) {
	isolatedEnv(t)
	// The test binary's version is the "dev" default, which effectiveVersion()
	// treats as 0.0.0 for comparison purposes.
	isolatedUpdateEnv(t, "v0.0.0")

	out, err := execCmd(t, "update")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out, "already on the latest version") {
		t.Fatalf("expected up-to-date message, got:\n%s", out)
	}
}

func TestUpdate_CheckOnly(t *testing.T) {
	isolatedEnv(t)
	isolatedUpdateEnv(t, "v99.0.0")

	out, err := execCmd(t, "update", "--check")
	if err != nil {
		t.Fatalf("update --check: %v", err)
	}
	if !strings.Contains(out, "v99.0.0") {
		t.Fatalf("expected the new version to be mentioned, got:\n%s", out)
	}
	if !strings.Contains(out, "Run 'kzgit update'") {
		t.Fatalf("expected --check to stop short of installing, got:\n%s", out)
	}
}

func TestUpdate_DeclinedPrompt(t *testing.T) {
	isolatedEnv(t)
	isolatedUpdateEnv(t, "v99.0.0")

	root := newRootCmd()
	root.SetArgs([]string{"update"})
	root.SetIn(strings.NewReader("n\n"))
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(buf.String(), "Update skipped") {
		t.Fatalf("expected the update to be skipped on 'n', got:\n%s", buf.String())
	}
}

func TestUpdate_YesFlag_InstallFails(t *testing.T) {
	isolatedEnv(t)
	isolatedUpdateEnv(t, "v99.0.0")

	// Point ReleaseBaseURL at a server with no matching assets, so Install
	// fails at the download step (well before it would ever touch the
	// running test binary via replaceExecutable).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	origBase := kzupdate.ReleaseBaseURL
	kzupdate.ReleaseBaseURL = srv.URL
	t.Cleanup(func() { kzupdate.ReleaseBaseURL = origBase })

	_, err := execCmd(t, "update", "--yes")
	if err == nil {
		t.Fatal("expected an error when the release asset can't be downloaded")
	}
	if !strings.Contains(err.Error(), "installing update") {
		t.Fatalf("expected an 'installing update' error, got: %v", err)
	}
}

func TestEffectiveVersionAndLabel_NonDev(t *testing.T) {
	orig := version
	version = "0.1.0"
	t.Cleanup(func() { version = orig })

	if got := effectiveVersion(); got != "0.1.0" {
		t.Fatalf("effectiveVersion() = %q, want %q", got, "0.1.0")
	}
	if got := versionLabel("0.1.0"); got != "v0.1.0" {
		t.Fatalf("versionLabel() = %q, want %q", got, "v0.1.0")
	}
}

func TestConfirm_NoInput(t *testing.T) {
	root := newRootCmd()
	root.SetIn(strings.NewReader(""))
	var buf strings.Builder
	root.SetOut(&buf)

	confirmed, err := confirm(root, "Proceed?")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed {
		t.Fatal("expected confirm to default to false on EOF/no input")
	}
}

func TestUpdate_NetworkError(t *testing.T) {
	isolatedEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	origAPI := kzupdate.APIURL
	kzupdate.APIURL = srv.URL
	t.Cleanup(func() { kzupdate.APIURL = origAPI })

	_, err := execCmd(t, "update")
	if err == nil {
		t.Fatal("expected an error when the release API is unreachable")
	}
}
