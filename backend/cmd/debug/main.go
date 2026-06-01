package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Check if migration file exists
	migrationsDir := "d:/Server panel/backend/migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		fmt.Printf("Error reading migrations dir: %v\n", err)
		return
	}
	
	fmt.Println("Migration files found:")
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if len(name) >= 6 && name[len(name)-6:] == ".up.sql" {
				fmt.Printf("  - %s\n", name)
				
				// Read file content
				content, err := os.ReadFile(filepath.Join(migrationsDir, name))
				if err != nil {
					fmt.Printf("    Error reading: %v\n", err)
					continue
				}
				fmt.Printf("    Size: %d bytes\n", len(content))
				
				// Check for created_at
				if len(content) > 500 {
					// Check around line 300 for backup table created_at
					for i := 0; i < len(content); i++ {
						if i+7 < len(content) && string(content[i:i+7]) == "created_" {
							if i+12 < len(content) && string(content[i:i+12]) == "created_at D" {
								fmt.Printf("    Found 'created_at D' at byte %d\n", i)
								break
							}
						}
					}
				}
			}
		}
	}
}