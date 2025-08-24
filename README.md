# LocIp - IP Geolocation Tool

A comprehensive IP geolocation tool written in Go that provides multiple lookup methods:

- **Local GeoLite2 Database**: Fast local lookups for city, region, country, and coordinates
- **ipinfo.io**: Online lookups with detailed information including organization and timezone
- **AbuseIPDB**: Abuse reputation checking for IP addresses

## Features

- Multiple lookup sources (local DB, ipinfo.io, AbuseIPDB)
- File processing support (batch IP lookups)
- Colored output for better readability
- Automatic API key detection
- Flexible command-line options

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd LocIp
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the binary:
```bash
go build -o locip .
```

## Usage

### Basic Commands

```bash
# Show help
./locip -h

# Query ipinfo.io for your public IP
./locip -i

# Query ipinfo.io for a specific IP
./locip -i 8.8.8.8

# Query AbuseIPDB for abuse information
./locip -a 1.2.3.4

# Use local GeoLite2 database for an IP
./locip 1.1.1.1

# Process a file with IP addresses
./locip my_ips.txt
```

### AbuseIPDB Options

```bash
# Basic abuse check
./locip -a 1.2.3.4

# With custom age (30 days)
./locip -a 1.2.3.4 --age 30

# Raw JSON output
./locip -a 1.2.3.4 --raw

# Combine options
./locip -a 1.2.3.4 --age 30 --raw
```

## Configuration

### AbuseIPDB API Key

Set your AbuseIPDB API key using one of these methods:

1. **Environment variable**:
```bash
export ABUSEIPDB_KEY="your_api_key_here"
```

2. **Configuration file**:
```bash
mkdir -p ~/.config/abuseipdb
echo "your_api_key_here" > ~/.config/abuseipdb/key
```

3. **Home directory file**:
```bash
echo "your_api_key_here" > ~/.abuseipdb_key
```

### GeoLite2 Database

The tool expects the GeoLite2 City database at `/opt/4rji/GeoLite2-City.mmdb`. You can:

1. Download it from MaxMind
2. Place it in the expected location
3. Or modify the `dbPath` constant in `geoip.go`

## Project Structure

The project is organized into modular files for better maintainability:

- **`locip.go`**: Main function and command-line logic
- **`geoip.go`**: GeoLite2 database operations
- **`ipinfo.go`**: ipinfo.io API integration
- **`abuseip.go`**: AbuseIPDB API integration
- **`utils.go`**: Common utility functions and file processing

## Dependencies

- `github.com/oschwald/geoip2-golang`: GeoLite2 database reader
- Standard Go libraries for HTTP, JSON, and file operations

## Examples

### Batch Processing

Create a file `ips.txt` with IP addresses (one per line):
```
8.8.8.8
1.1.1.1
208.67.222.222
```

Then process it:
```bash
./locip ips.txt
```

### Abuse Check with Custom Age

```bash
./locip -a 192.168.1.1 --age 60
```

This will check the abuse reputation for the last 60 days instead of the default 90 days.

## License

[Add your license information here] 