package main

import (
	"fmt"
	"net"
	"os"

	"github.com/oschwald/geoip2-golang"
)

const dbPath = "/opt/4rji/GeoLite2-City.mmdb"

func checkDatabase() {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("Warning: GeoLite2 database not found at %s. Local lookup features will be unavailable.\n", dbPath)
		// Do not exit, allow the program to continue for -i option or other fallbacks.
	}
}

func printFullRecord(ipStr string) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		fmt.Printf("Error: Could not access local GeoLite2 database to look up %s.\n", ipStr)
		fmt.Printf("Reason: %v\n", err)
		fmt.Printf("Please ensure the database file exists at %s and is valid.\n", dbPath)
		fmt.Printf("Alternatively, try 'locip -i %s' for an online lookup.\n", ipStr)
		return
	}
	defer db.Close()

	ip := net.ParseIP(ipStr)
	record, err := db.City(ip)
	if err != nil {
		fmt.Printf("Error: Could not retrieve GeoIP details for %s from the local database.\n", ipStr)
		fmt.Printf("Reason: %v\n", err)
		fmt.Printf("Consider using 'locip -i %s' for an alternative online lookup.\n", ipStr)
		return
	}

	region := "Unknown"
	if len(record.Subdivisions) > 0 {
		region = record.Subdivisions[0].Names["en"]
	}
	fmt.Printf("[*] Target: %s Geo-located.\n", ipStr)
	fmt.Printf("[+] %s, %s, %s\n", record.City.Names["en"], region, record.Country.Names["en"])
	fmt.Printf("[+] Latitude: %f, Longitude: %f\n", record.Location.Latitude, record.Location.Longitude)
}

func printCityOnly(ipStr string, db *geoip2.Reader) {
	ip := net.ParseIP(ipStr)
	record, err := db.City(ip)
	if err != nil {
		fmt.Printf("[!] Could not retrieve city details for %s from local database: %v\n", ipStr, err)
		fmt.Printf("[+] %s -> Error looking up in local DB\n", ipStr)
		return
	}

	city := record.City.Names["en"]
	region := "Unknown"
	if len(record.Subdivisions) > 0 {
		region = record.Subdivisions[0].Names["en"]
	}
	country := record.Country.Names["en"]

	if city == "" && region == "" && country == "" {
		fmt.Printf("[+] %s -> Information not available in local DB\n", ipStr)
	} else {
		fmt.Printf("[+] %s -> %s, %s, %s\n", ipStr, city, region, country)
	}
}
