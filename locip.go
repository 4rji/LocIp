package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/oschwald/geoip2-golang"
)

const defaultDBPath = "/opt/4rji/GeoLite2-City.mmdb"
const outputSeparator = "--------------------------------------------------"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

type colorizer struct {
	enabled bool
}

func (c colorizer) sprint(color, text string) string {
	if !c.enabled {
		return text
	}
	return color + text + colorReset
}

func (c colorizer) label(text string) string {
	return c.sprint(colorBlue, text)
}

func (c colorizer) ok(text string) string {
	return c.sprint(colorYellow, text)
}

func (c colorizer) warn(text string) string {
	return c.sprint(colorYellow, text)
}

func (c colorizer) err(text string) string {
	return c.sprint(colorRed, text)
}

func (c colorizer) info(text string) string {
	return c.sprint(colorBlue, text)
}

type config struct {
	dbPath  string
	localDB bool
	online  bool
	noColor bool
	targets []string
}

type ipInfo struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Readme   string `json:"readme"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, help, err := parseArgs(args, stderr)
	colors := colorizer{enabled: shouldUseColor(cfg.noColor)}
	if err != nil {
		fmt.Fprintf(stderr, "%s %v\n\n", colors.err("Error:"), err)
		printUsage(stderr, colors)
		return 2
	}
	if help {
		printUsage(stdout, colors)
		return 0
	}
	if len(cfg.targets) == 0 {
		printUsage(stdout, colors)
		return 0
	}

	target := cfg.targets[0]
	if !cfg.localDB {
		client := &http.Client{Timeout: 8 * time.Second}
		if isExistingFile(target) {
			return processIPInfoFile(stdout, stderr, client, target, colors)
		}

		if err := queryIPInfo(stdout, client, cfg.onlineTarget(), colors); err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colors.err("Error:"), err)
			return 1
		}
		return 0
	}

	if isExistingFile(target) {
		return processIPFile(stdout, stderr, target, cfg.dbPath, colors)
	}

	if err := printFullRecord(stdout, target, cfg.dbPath, colors); err != nil {
		fmt.Fprintf(stderr, "%s %v\n", colors.err("Error:"), err)
		return 1
	}
	return 0
}

func parseArgs(args []string, stderr io.Writer) (config, bool, error) {
	cfg := config{dbPath: defaultDBPath}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		return cfg, true, nil
	}

	fs := flag.NewFlagSet("locip", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {}
	fs.BoolVar(&cfg.localDB, "d", false, "use the local GeoLite2 database")
	fs.BoolVar(&cfg.online, "i", false, "query ipinfo.io; kept as a compatibility alias")
	fs.StringVar(&cfg.dbPath, "db", cfg.dbPath, "path to the GeoLite2 City database")
	fs.BoolVar(&cfg.noColor, "no-color", false, "disable ANSI color output")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return cfg, true, nil
		}
		return cfg, false, err
	}

	cfg.targets = fs.Args()
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "db" {
			cfg.localDB = true
		}
	})
	if cfg.online && cfg.localDB {
		return cfg, false, fmt.Errorf("use either -i for ipinfo.io or -d/-db for the local database, not both")
	}
	if !cfg.localDB && len(cfg.targets) > 1 {
		return cfg, false, fmt.Errorf("expected at most one IP address or hostname")
	}
	if cfg.localDB && len(cfg.targets) > 1 {
		return cfg, false, fmt.Errorf("expected at most one target")
	}
	return cfg, false, nil
}

func (c config) onlineTarget() string {
	if len(c.targets) == 0 {
		return ""
	}
	return c.targets[0]
}

func shouldUseColor(disabled bool) bool {
	if disabled {
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return true
}

func printUsage(w io.Writer, colors colorizer) {
	fmt.Fprintf(w, "%s locip [options] [target]\n", colors.label("Usage:"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Looks up geolocation information for IP addresses using a local GeoLite2 database or ipinfo.io.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", colors.label("Options:"))
	fmt.Fprintln(w, "  -d                Use the local GeoLite2 database at the default path.")
	fmt.Fprintln(w, "  -db <path>        Use the local GeoLite2 database at a custom path.")
	fmt.Fprintln(w, "  -i                Query ipinfo.io explicitly. This is the default behavior.")
	fmt.Fprintln(w, "  -no-color         Disable ANSI color output.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", colors.label("Targets:"))
	fmt.Fprintln(w, "  <ip_address>      Query ipinfo.io by default.")
	fmt.Fprintln(w, "  <filepath>        Process a file using ipinfo.io by default.")
	fmt.Fprintln(w, "  -d <ip_address>   Use the local GeoLite2 database to locate an IP address.")
	fmt.Fprintln(w, "  -d <filepath>     Process a file containing one IP address per line.")
	fmt.Fprintln(w, "  no arguments      Show this help.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", colors.label("Examples:"))
	fmt.Fprintln(w, "  locip 8.8.8.8")
	fmt.Fprintln(w, "  locip ips.txt")
	fmt.Fprintln(w, "  locip -d 1.1.1.1")
	fmt.Fprintln(w, "  locip -d my_ip_list.txt")
	fmt.Fprintln(w, "  locip -db ./GeoLite2-City.mmdb 1.1.1.1")
}

func printFullRecord(stdout io.Writer, ipStr, dbPath string, colors colorizer) error {
	ip, err := parseIP(ipStr)
	if err != nil {
		return err
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		return fmt.Errorf("could not access local GeoLite2 database at %q: %w\ntry %q for an online lookup", dbPath, err, "locip "+ipStr)
	}
	defer db.Close()

	record, err := db.City(ip)
	if err != nil {
		return fmt.Errorf("could not retrieve GeoIP details for %s from the local database: %w", ipStr, err)
	}

	city := valueOrUnknown(record.City.Names["en"])
	region := "Unknown"
	if len(record.Subdivisions) > 0 {
		region = valueOrUnknown(record.Subdivisions[0].Names["en"])
	}
	country := valueOrUnknown(record.Country.Names["en"])

	fmt.Fprintf(stdout, "%s %s %s\n", colors.info("[*]"), colors.label("Target:"), colors.ok(ipStr+" geo-located"))
	fmt.Fprintf(stdout, "%s %s, %s, %s\n", colors.ok("[+]"), city, region, country)
	fmt.Fprintf(stdout, "%s %s %.6f, %s %.6f\n", colors.ok("[+]"), colors.label("Latitude:"), record.Location.Latitude, colors.label("Longitude:"), record.Location.Longitude)
	return nil
}

func printCityOnly(stdout io.Writer, ipStr string, db *geoip2.Reader, colors colorizer) {
	ip, err := parseIP(ipStr)
	if err != nil {
		fmt.Fprintf(stdout, "%s %s -> %v\n", colors.warn("[!]"), ipStr, err)
		return
	}

	record, err := db.City(ip)
	if err != nil {
		fmt.Fprintf(stdout, "%s %s -> error looking up in local DB: %v\n", colors.warn("[!]"), ipStr, err)
		return
	}

	city := record.City.Names["en"]
	region := ""
	if len(record.Subdivisions) > 0 {
		region = record.Subdivisions[0].Names["en"]
	}
	country := record.Country.Names["en"]

	location := joinNonEmpty(city, region, country)
	if location == "" {
		fmt.Fprintf(stdout, "%s %s -> information not available in local DB\n", colors.warn("[!]"), ipStr)
		return
	}
	fmt.Fprintf(stdout, "%s %s -> %s\n", colors.ok("[+]"), ipStr, location)
}

func queryIPInfo(stdout io.Writer, client *http.Client, target string, colors colorizer) error {
	reqURL, err := ipInfoURL(target)
	if err != nil {
		return err
	}

	resp, err := client.Get(reqURL)
	if err != nil {
		return fmt.Errorf("fetching IP info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("ipinfo.io returned %s", resp.Status)
	}

	var info ipInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("decoding ipinfo.io JSON: %w", err)
	}

	printIPInfo(stdout, info, colors)
	return nil
}

func ipInfoURL(target string) (string, error) {
	base := "https://ipinfo.io/"
	target = strings.TrimSpace(target)
	if target == "" {
		return base + "json", nil
	}
	if strings.ContainsAny(target, "/?#") {
		return "", fmt.Errorf("invalid ipinfo target %q", target)
	}
	return base + url.PathEscape(target) + "/json", nil
}

func printIPInfo(stdout io.Writer, info ipInfo, colors colorizer) {
	printField(stdout, colors, "IP", info.IP)
	printField(stdout, colors, "Hostname", info.Hostname)
	printField(stdout, colors, "City", info.City)
	printField(stdout, colors, "Region", info.Region)
	printField(stdout, colors, "Country", info.Country)
	printField(stdout, colors, "Location", info.Loc)
	printField(stdout, colors, "Organization", info.Org)
	printField(stdout, colors, "Postal Code", info.Postal)
	printField(stdout, colors, "Timezone", info.Timezone)
}

func printField(stdout io.Writer, colors colorizer, label, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(stdout, "%s %s\n", colors.label(label+":"), value)
}

func processIPFile(stdout, stderr io.Writer, filePath, dbPath string, colors colorizer) int {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "%s cannot process %q using the local GeoLite2 database at %q: %v\n", colors.err("Error:"), filePath, dbPath, err)
		fmt.Fprintf(stderr, "%s use -db to set a database path or run without -d for an online lookup.\n", colors.warn("Hint:"))
		return 1
	}
	defer db.Close()

	file, err := os.Open(filePath)
	if err != nil {
		if filePath == "ips.txt" && os.IsNotExist(err) {
			fmt.Fprintf(stdout, "%s default file %q not found.\n\n", colors.warn("Warning:"), filePath)
			printUsage(stdout, colors)
			return 0
		}
		fmt.Fprintf(stderr, "%s could not open IP list file %q: %v\n", colors.err("Error:"), filePath, err)
		return 1
	}
	defer file.Close()

	fmt.Fprintf(stdout, "%s Processing IPs from file: %s\n", colors.info("[*]"), colors.sprint(colorBlue, filePath))
	scanner := bufio.NewScanner(file)
	foundIPs := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if foundIPs {
			printSeparator(stdout, colors)
		}
		printCityOnly(stdout, line, db, colors)
		foundIPs = true
	}
	if !foundIPs {
		fmt.Fprintf(stdout, "%s no IP addresses found in %s.\n", colors.warn("Warning:"), filePath)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "%s could not read IP list file %q: %v\n", colors.err("Error:"), filePath, err)
		return 1
	}
	return 0
}

func processIPInfoFile(stdout, stderr io.Writer, client *http.Client, filePath string, colors colorizer) int {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "%s could not open IP list file %q: %v\n", colors.err("Error:"), filePath, err)
		return 1
	}
	defer file.Close()

	fmt.Fprintf(stdout, "%s Processing IPs from file: %s (using ipinfo.io)\n", colors.info("[*]"), colors.sprint(colorBlue, filePath))
	scanner := bufio.NewScanner(file)
	foundIPs := false
	hadErrors := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if foundIPs {
			printSeparator(stdout, colors)
		}
		foundIPs = true
		fmt.Fprintf(stdout, "%s %s\n", colors.info("[*]"), colors.label(line))
		if err := queryIPInfo(stdout, client, line, colors); err != nil {
			hadErrors = true
			fmt.Fprintf(stderr, "%s %s -> %v\n", colors.err("Error:"), line, err)
		}
	}
	if !foundIPs {
		fmt.Fprintf(stdout, "%s no IP addresses found in %s.\n", colors.warn("Warning:"), filePath)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "%s could not read IP list file %q: %v\n", colors.err("Error:"), filePath, err)
		return 1
	}
	if hadErrors {
		return 1
	}
	return 0
}

func parseIP(ipStr string) (net.IP, error) {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address %q", ipStr)
	}
	return ip, nil
}

func isExistingFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func printSeparator(stdout io.Writer, colors colorizer) {
	fmt.Fprintln(stdout, colors.sprint(colorBlue, outputSeparator))
}

func joinNonEmpty(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ", ")
}

func valueOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	return value
}
