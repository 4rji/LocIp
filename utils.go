package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

func printUsage() {
	fmt.Println("Usage: locip [options] [target]")
	fmt.Println("\nLooks up geolocation information for IP addresses using a local GeoLite2 database, ipinfo.io, or AbuseIPDB.")
	fmt.Println("\nOptions:")
	fmt.Println("  -i [ip_address]   Query ipinfo.io for the given IP address (or your public IP if none provided).")
	fmt.Println("                    Displays detailed information including city, region, country, location, etc.")
	fmt.Println("  -a <ip_address>   Query AbuseIPDB for abuse information about the given IP address.")
	fmt.Println("                    Options: --age <days> (default: 90), --raw (JSON output)")
	fmt.Println("  -a <filepath>     Query AbuseIPDB for abuse information about each IP in the specified file.")
	fmt.Println("                    Each IP should be on a separate line. Supports --age and --raw options.")
	fmt.Println("\nTargets (Uses local GeoLite2 Database):")
	fmt.Println("  <ip_address>      Show full geolocation details (city, region, country, lat/long) for the given IP address.")
	fmt.Println("  <filepath>        Process a file containing a list of IP addresses (one per line).")
	fmt.Println("                    For each IP, shows city, region, and country.")
	fmt.Println("\nDefault Behavior (Uses local GeoLite2 Database):")
	fmt.Println("  If no arguments are provided, the script attempts to read and process 'ips.txt'")
	fmt.Println("  from the current directory. It expects one IP address per line and will show")
	fmt.Println("  city, region, and country for each.")
	fmt.Println("\nExamples:")
	fmt.Println("  locip -i 8.8.8.8       # Query ipinfo.io for 8.8.8.8")
	fmt.Println("  locip -i               # Query ipinfo.io for your public IP")
	fmt.Println("  locip -a 1.2.3.4       # Query AbuseIPDB for 1.2.3.4")
	fmt.Println("  locip -a 1.2.3.4 --age 30 --raw  # Query with custom age and raw JSON output")
	fmt.Println("  locip -a ip_list.txt    # Query AbuseIPDB for each IP in ip_list.txt")
	fmt.Println("  locip 1.1.1.1          # Use local DB for full details of 1.1.1.1")
	fmt.Println("  locip my_ip_list.txt   # Use local DB to process IPs in my_ip_list.txt")
	fmt.Println("  locip                  # Use local DB to process IPs in ips.txt (if it exists)")
	fmt.Println("  locipinst              # Run this command to install/update the GeoLite2 database (if locipinst script is available)")
}

// processIPFile reads a file containing IP addresses (one per line)
// and prints city-only information for each valid IP using the local GeoLite2 database.
func processIPFile(filePath string) {
	// Attempt to open the database ONCE for the entire file processing.
	db, errDB := geoip2.Open(dbPath)
	if errDB != nil {
		fmt.Printf("Error: Cannot process file '%s' using the local GeoLite2 database.\n", filePath)
		fmt.Printf("Reason: Failed to open database at '%s': %v\n", dbPath, errDB)
		fmt.Println("Please ensure the database file exists and is valid, or use the '-i <ip>' option for individual online lookups.")
		return // Stop processing this file if DB can't be opened.
	}
	defer db.Close()

	file, err := os.Open(filePath)
	if err != nil {
		if filePath == "ips.txt" && os.IsNotExist(err) {
			fmt.Printf("Default file '%s' not found.\n", filePath)
			printUsage() // This already exits if it's the default file and not found.
			return       // Return to be safe, though printUsage exits.
		}
		fmt.Printf("Error: Could not open IP list file '%s': %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Printf("[*] Processing IPs from file: %s (using local DB)\n", filePath)
	scanner := bufio.NewScanner(file)
	foundIPs := false
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip != "" {
			printCityOnly(ip, db) // Pass the opened db
			foundIPs = true
		}
	}
	if !foundIPs {
		fmt.Printf("No IP addresses found in %s.\n", filePath)
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error: Could not read IP list file '%s': %v\n", filePath, err)
		os.Exit(1)
	}
}

// processAbuseIPFile reads a file containing IP addresses (one per line)
// and checks each IP using AbuseIPDB with the specified parameters.
func processAbuseIPFile(filePath string, maxAge int, raw bool) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error: Could not open IP list file '%s': %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Printf("[*] Processing IPs from file: %s (using AbuseIPDB)\n", filePath)
	scanner := bufio.NewScanner(file)
	foundIPs := false
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip != "" {
			fmt.Printf("\n--- Checking IP: %s ---\n", ip)
			checkAbuseIP(ip, maxAge, raw)
			foundIPs = true
		}
	}
	if !foundIPs {
		fmt.Printf("No IP addresses found in %s.\n", filePath)
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error: Could not read IP list file '%s': %v\n", filePath, err)
		os.Exit(1)
	}
}
