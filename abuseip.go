package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const abuseBaseURL = "https://api.abuseipdb.com/api/v2/check"

// AbuseIPDB response structure
type abuseCheckResp struct {
	Data struct {
		IPAddress            string      `json:"ipAddress"`
		AbuseConfidenceScore int         `json:"abuseConfidenceScore"`
		TotalReports         int         `json:"totalReports"`
		LastReportedAt       *string     `json:"lastReportedAt"`
		UsageType            *string     `json:"usageType"`
		Domain               *string     `json:"domain"`
		CountryCode          *string     `json:"countryCode"`
		Isp                  *string     `json:"isp"`
		Reports              interface{} `json:"reports,omitempty"`
	} `json:"data"`
}

// AbuseIPDB helper functions
func getAbuseAPIKey() (string, error) {
	if v := os.Getenv("ABUSEIPDB_KEY"); v != "" {
		return v, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".config", "abuseipdb", "key"),
		filepath.Join(home, ".abuseipdb_key"),
	}
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil {
			return string(trimNL(b)), nil
		}
	}
	return "", errors.New("no key")
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}

func nz(s *string, def string) string {
	if s == nil || *s == "" {
		return def
	}
	return *s
}

func checkAbuseIP(ip string, maxAge int, raw bool) {
	key, err := getAbuseAPIKey()
	if err != nil || key == "" {
		fmt.Fprintf(os.Stderr, "ERR: ABUSEIPDB_KEY no definido\n")
		os.Exit(1)
	}

	cl := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", abuseBaseURL, nil)
	q := req.URL.Query()
	q.Add("ipAddress", ip)
	q.Add("maxAgeInDays", fmt.Sprintf("%d", maxAge))
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Key", key)
	req.Header.Set("Accept", "application/json")

	resp, err := cl.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERR: %s\n", err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		fmt.Fprintf(os.Stderr, "ERR: HTTP %d: %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}

	var out abuseCheckResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		fmt.Fprintf(os.Stderr, "ERR: %s\n", err.Error())
		os.Exit(1)
	}

	if raw {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out.Data)
		return
	}

	fmt.Printf("%s\tabuseScore=%d\treports=%d\tlastSeen=%s\n",
		out.Data.IPAddress,
		out.Data.AbuseConfidenceScore,
		out.Data.TotalReports,
		nz(out.Data.LastReportedAt, "-"),
	)
}
