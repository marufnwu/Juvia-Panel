package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	migrationsDir := flag.String("migrations", "/etc/panel/migrations", "Path to migrations directory")
	flag.Parse()

	// If the default path doesn't exist, try common alternatives
	candidates := []string{
		*migrationsDir,
		"/etc/panel/migrations",
		"/var/panel/migrations",
		"migrations",
		"../migrations",
		"../../migrations",
	}

	var found string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			found = c
			break
		}
	}
	if found == "" {
		fmt.Printf("Error: no migrations directory found in: %v\n", candidates)
		fmt.Println("Use --migrations flag to specify the path")
		os.Exit(1)
	}

	entries, err := os.ReadDir(found)
	if err != nil {
		fmt.Printf("Error reading migrations dir %s: %v\n", found, err)
		os.Exit(1)
	}

	fmt.Printf("Migration files found in %s:\n", found)
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if len(name) >= 7 && name[len(name)-7:] == ".up.sql" {
				fmt.Printf("  - %s\n", name)
			}
		}
	}
	_ = filepath.Join
}
