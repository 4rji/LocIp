package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// ANSI escape codes for colors
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

type IPInfo struct {
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

func queryIpInfo(ipAddress string) {
	url := "https://ipinfo.io/"
	if ipAddress != "" {
		url += ipAddress + "/"
	}
	url += "json"

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching IP info: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var ipInfo IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		log.Printf("Error decoding IP info JSON: %v\n", err)
		return
	}

	fmt.Println() // Línea extra
	fmt.Printf("%sIP:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.IP, ColorReset)
	if ipInfo.City != "" {
		fmt.Printf("%sCity:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.City, ColorReset)
	}
	if ipInfo.Region != "" {
		fmt.Printf("%sRegion:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.Region, ColorReset)
	}
	if ipInfo.Country != "" {
		fmt.Printf("%sCountry:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.Country, ColorReset)
	}
	if ipInfo.Loc != "" {
		fmt.Printf("%sLocation:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.Loc, ColorReset)
	}
	if ipInfo.Org != "" {
		fmt.Printf("%sOrganization:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.Org, ColorReset)
	}
	if ipInfo.Postal != "" {
		fmt.Printf("%sPostal Code:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.Postal, ColorReset)
	}
	if ipInfo.Timezone != "" {
		fmt.Printf("%sTimezone:%s %s%s%s\n", ColorBlue, ColorReset, ColorYellow, ipInfo.Timezone, ColorReset)
	}
}
