# LocIp

This project uses Go and the `geoip2-golang` library to perform IP geolocation.

## Setup

To initialize the project and download the necessary dependencies, run the following commands:

```bash
go mod init locip
go get github.com/oschwald/geoip2-golang
```

## Usage

```
Usage: locip [options] [target]

Looks up geolocation information for IP addresses using a local GeoLite2 database or ipinfo.io.

Options:
  -i [ip_address]   Query ipinfo.io for the given IP address (or your public IP if none provided).
                    Displays detailed information including city, region, country, location, etc.

Targets (Uses local GeoLite2 Database):
  <ip_address>      Show full geolocation details (city, region, country, lat/long) for the given IP address.
  <filepath>        Process a file containing a list of IP addresses (one per line).
                    For each IP, shows city, region, and country.

Default Behavior (Uses local GeoLite2 Database):
  If no arguments are provided, the script attempts to read and process 'ips.txt'
  from the current directory. It expects one IP address per line and will show
  city, region, and country for each.

Examples:
  locip -i 8.8.8.8       # Query ipinfo.io for 8.8.8.8
  locip -i               # Query ipinfo.io for your public IP
  locip 1.1.1.1          # Use local DB for full details of 1.1.1.1
  locip my_ip_list.txt   # Use local DB to process IPs in my_ip_list.txt
  locip                  # Use local DB to process IPs in ips.txt (if it exists)
  locipinst              # Run this command to install/update the GeoLite2 database (if locipinst script is available) 