package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
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

func TestParseArgsRejectsOnlineAlias(t *testing.T) {
	t.Parallel()

	_, _, err := parseArgs([]string{"-i", "8.8.8.8"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected -i to be rejected")
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

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Status:     "429 Too Many Requests",
			Body:       io.NopCloser(strings.NewReader("rate limited")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

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

func TestRunWithoutArgsQueriesCurrentIP(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/json" {
			t.Errorf("unexpected path: %s", req.URL.Path)
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`{"ip":"203.0.113.10","city":"Example City","org":"AS64496 Example ISP"}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	t.Cleanup(func() {
		http.DefaultTransport = oldTransport
	})

	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run returned %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "203.0.113.10") {
		t.Fatalf("stdout = %q, want current IP lookup result", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Tip: use -h for more options.") {
		t.Fatalf("stdout = %q, want startup tip", stdout.String())
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

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/1.1.1.1/json" && req.URL.Path != "/8.8.8.8/json" {
			t.Errorf("unexpected path: %s", req.URL.Path)
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`{"ip":"1.1.1.1","org":"AS13335 Cloudflare, Inc."}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
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
