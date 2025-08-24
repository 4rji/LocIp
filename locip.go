package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		// No arguments: Default behavior is to process ips.txt.
		// BUT, if the GeoLite2 DB is missing, just show usage and exit cleanly.
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			printUsage()
			os.Exit(0) // Exit cleanly with status 0, no warnings/errors.
		}
		// If we are here, it means no args were given AND the DB likely exists (or os.Stat had a different error).
		// Proceed with normal DB check and default file processing.
		checkDatabase() // This will be silent if DB exists, or print warning for other DB issues.
		processIPFile("ips.txt")
		return
	}

	// If arguments ARE provided, it's okay for checkDatabase() to print a warning
	// if the DB is missing, as local DB operations might be intended by the user.
	checkDatabase()

	firstArg := args[0]

	if firstArg == "-h" || firstArg == "--help" {
		printUsage()
		return
	}

	if firstArg == "-i" {
		ipToQuery := ""
		if len(args) > 1 {
			// Check if the next argument is another flag or an actual IP/domain
			if !strings.HasPrefix(args[1], "-") {
				ipToQuery = args[1]
			} else {
				// -i was given, but next arg looks like another flag, so query self IP
				// and then it will be an invalid arg combination by printUsage
			}
		}
		queryIpInfo(ipToQuery) // queryIpInfo handles empty string for self-IP
		// If -i was followed by more than one non-flag argument, or a flag after IP
		if (ipToQuery != "" && len(args) > 2) || (ipToQuery == "" && len(args) > 1) {
			// Example: locip -i 8.8.8.8 something_else OR locip -i -h
			if !(ipToQuery != "" && len(args) == 2 && (args[1] == "-h" || args[1] == "--help")) { // allow locip -i ip -h
				if !(ipToQuery == "" && len(args) == 1 && (args[0] == "-h" || args[0] == "--help")) { // allow locip -i -h
					fmt.Println("\nWarning: Extra arguments provided with -i option. Processing -i and ignoring others, or use -h for help.")
					// If it was "locip -i someotherflag", it would be an error.
					// If it was "locip -i ip someotherflag", it is also an error.
					if (ipToQuery == "" && len(args) > 1) || (ipToQuery != "" && len(args) > 2) {
						printUsage()
						os.Exit(1)
					}
				}
			}
		}
		return
	}

	if firstArg == "-a" {
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "ERR: falta IP después de -a\n")
			printUsage()
			os.Exit(1)
		}
		ipToCheck := args[1]
		maxAge := 90
		raw := false

		// Check for additional flags
		for i := 2; i < len(args); i++ {
			if args[i] == "--age" && i+1 < len(args) {
				if age, err := fmt.Sscanf(args[i+1], "%d", &maxAge); err != nil || age != 1 {
					fmt.Fprintf(os.Stderr, "ERR: valor inválido para --age\n")
					os.Exit(1)
				}
				i++ // Skip the next argument
			} else if args[i] == "--raw" {
				raw = true
			} else {
				fmt.Fprintf(os.Stderr, "ERR: argumento desconocido: %s\n", args[i])
				printUsage()
				os.Exit(1)
			}
		}

		checkAbuseIP(ipToCheck, maxAge, raw)
		return
	}

	// At this point, not -i, not -h, not --help, and not 0 arguments.
	// It must be a single argument: either an IP or a filepath for GeoLite2.
	if len(args) == 1 {
		target := args[0]
		// Check if the argument is a file that exists and is not a directory.
		if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
			processIPFile(target) // Process the specified file using local GeoLite2
		} else {
			// Treat as a single IP for full record using local GeoLite2
			// We can add a simple IP validation here if needed, but GeoLite2 will error out anyway.
			printFullRecord(target)
		}
		return
	}

	// If we reach here, it's an invalid combination of arguments.
	fmt.Println("Invalid arguments or combination.")
	printUsage()
	os.Exit(1)
}
