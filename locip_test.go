package main

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestParseArgsDefaultsToOnlineTarget(t *testing.T) {
	t.Parallel()

	cfg, help, err := parseArgs([]string{"8.8.8.8"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if help {
		t.Fatal("parseArgs returned help=true")
	}
	if cfg.localDB {
		t.Fatal("expected default mode to use ipinfo.io")
	}
	if got := cfg.onlineTarget(); got != "8.8.8.8" {
		t.Fatalf("onlineTarget() = %q, want %q", got, "8.8.8.8")
	}
}

func TestParseArgsOnlineAlias(t *testing.T) {
	t.Parallel()

	cfg, help, err := parseArgs([]string{"-i", "8.8.8.8"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if help {
		t.Fatal("parseArgs returned help=true")
	}
	if !cfg.online {
		t.Fatal("expected -i compatibility alias to be tracked")
	}
	if cfg.localDB {
		t.Fatal("did not expect local DB mode")
	}
}

func TestParseArgsLocalDBDefaultPath(t *testing.T) {
	t.Parallel()

	cfg, help, err := parseArgs([]string{"-d", "1.1.1.1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if help {
		t.Fatal("parseArgs returned help=true")
	}
	if !cfg.localDB {
		t.Fatal("expected local DB mode")
	}
	if cfg.dbPath != defaultDBPath {
		t.Fatalf("dbPath = %q, want %q", cfg.dbPath, defaultDBPath)
	}
}

func TestParseArgsCustomDBImpliesLocalDB(t *testing.T) {
	t.Parallel()

	cfg, help, err := parseArgs([]string{"-db", "/tmp/custom.mmdb", "1.1.1.1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if help {
		t.Fatal("parseArgs returned help=true")
	}
	if !cfg.localDB {
		t.Fatal("expected -db to imply local DB mode")
	}
	if cfg.dbPath != "/tmp/custom.mmdb" {
		t.Fatalf("dbPath = %q, want %q", cfg.dbPath, "/tmp/custom.mmdb")
	}
}

func TestParseArgsRejectsExtraTargets(t *testing.T) {
	t.Parallel()

	_, _, err := parseArgs([]string{"1.1.1.1", "8.8.8.8"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for extra targets")
	}
}

func TestParseIP(t *testing.T) {
	t.Parallel()

	ip, err := parseIP("1.1.1.1")
	if err != nil {
		t.Fatalf("parseIP returned error: %v", err)
	}
	if !ip.Equal(net.ParseIP("1.1.1.1")) {
		t.Fatalf("parseIP returned %v", ip)
	}

	if _, err := parseIP("not-an-ip"); err == nil {
		t.Fatal("expected invalid IP error")
	}
}

func TestIPInfoURL(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":        "https://ipinfo.io/json",
		"8.8.8.8": "https://ipinfo.io/8.8.8.8/json",
	}
	for target, want := range tests {
		got, err := ipInfoURL(target)
		if err != nil {
			t.Fatalf("ipInfoURL(%q) returned error: %v", target, err)
		}
		if got != want {
			t.Fatalf("ipInfoURL(%q) = %q, want %q", target, got, want)
		}
	}

	if _, err := ipInfoURL("bad/path"); err == nil {
		t.Fatal("expected invalid target error")
	}
}

func TestQueryIPInfoHandlesHTTPStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := server.Client()
	oldTransport := http.DefaultTransport
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(server.URL, "http://")
		return oldTransport.RoundTrip(req)
	})

	var out bytes.Buffer
	err := queryIPInfo(&out, client, "8.8.8.8", colorizer{})
	if err == nil {
		t.Fatal("expected status error")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("error = %q, want it to mention status 429", err.Error())
	}
}

func TestRunHelpDisablesColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-h"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run returned %d, want 0; stderr=%q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "\033[") {
		t.Fatalf("help output contains ANSI escape codes: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("help output missing usage: %q", stdout.String())
	}
}

func TestRunWithoutArgsShowsHelp(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run returned %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout missing usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunExistingFileUsesIPInfoMode(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	file := t.TempDir() + "/ips.txt"
	if err := os.WriteFile(file, []byte("1.1.1.1\n8.8.8.8\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.1.1.1/json" && r.URL.Path != "/8.8.8.8/json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ip":"1.1.1.1","org":"AS13335 Cloudflare, Inc."}`))
	}))
	defer server.Close()

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(server.URL, "http://")
		return oldTransport.RoundTrip(req)
	})
	t.Cleanup(func() {
		http.DefaultTransport = oldTransport
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{file}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run returned %d, want 0; stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "using ipinfo.io") {
		t.Fatalf("stdout = %q, want ipinfo file mode", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Cloudflare") {
		t.Fatalf("stdout = %q, want ipinfo result", stdout.String())
	}
	if !strings.Contains(stdout.String(), outputSeparator) {
		t.Fatalf("stdout = %q, want separator between IPs", stdout.String())
	}
}

func TestShouldUseColorHonorsNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if shouldUseColor(false) {
		t.Fatal("expected color to be disabled when NO_COLOR is set")
	}
	if shouldUseColor(true) {
		t.Fatal("expected color to be disabled by explicit flag")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
