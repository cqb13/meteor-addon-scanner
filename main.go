package main

import (
	"dev/cqb13/meteor-addon-scanner/scanner"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MinimumMinecraftVersion *string  `json:"minimum_minecraft_version"`
	RepoBlacklist           []string `json:"repo-blacklist"`
	DeveloperBlacklist      []string `json:"developer-blacklist"`
	Verified                []string `json:"verified"`
	IgnoreArchived          bool     `json:"ignore_archived"`
	IgnoreForks             bool     `json:"ignore_forks"`
}

type Stats struct {
	LastUpdated          string  `json:"last_updated"`
	ValidAddonsCount     int     `json:"valid_addons_count"`
	ArchivedAddonsCount  int     `json:"archived_addons_count"`
	InvalidAddonsCount   int     `json:"invalid_addons_count"`
	ExecutionTimeSeconds float64 `json:"execution_time_seconds"`
}

func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}

	return &config, nil
}

func validateOutputPath(output string) error {
	if !strings.HasSuffix(output, ".json") {
		return fmt.Errorf("Output path must lead to a json file")
	}

	if _, err := os.Stat(output); err == nil {
		return fmt.Errorf("Output path already exists")
	}

	return nil
}

func main() {
	startTime := time.Now()
	args := os.Args

	if len(args) < 2 {
		fmt.Println("Usage: meteor-addon-scanner <output.json>")
		return
	}

	outputPath := args[1]

	err := validateOutputPath(outputPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Load config.json
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Printf("Failed to load config: %s\n", err)
		return
	}

	// Load .env file
	if _, err := os.Stat(".env"); err == nil {
		err = godotenv.Load()
		if err != nil {
			fmt.Println("Failed to load env file:", err)
			return
		}
	} else {
		fmt.Println(".env file not found, assuming environment variable is set externally")
	}

	var key string = os.Getenv("KEY")
	scanner.InitDefaultHeaders(key)

	fmt.Println("Locating Repositories")
	repos := scanner.Locate(config.Verified)
	fmt.Printf("Located %v repos\n", len(repos))

	// Apply repo blacklist
	removed := 0
	for _, repo := range config.RepoBlacklist {
		_, ok := repos[repo]
		if ok {
			delete(repos, repo)
			removed++
		}
	}
	fmt.Printf("Removed %d/%d repo blacklisted repositories\n", removed, len(config.RepoBlacklist))

	// Apply developer blacklist
	developerRemoved := 0
	for repoName := range repos {
		// Extract owner from "owner/repo" format
		parts := strings.Split(repoName, "/")
		if len(parts) == 2 {
			owner := strings.ToLower(parts[0])
			for _, blacklistedDev := range config.DeveloperBlacklist {
				if owner == strings.ToLower(blacklistedDev) {
					delete(repos, repoName)
					developerRemoved++
					break
				}
			}
		}
	}
	fmt.Printf("Removed %d repositories from blacklisted developers\n", developerRemoved)

	fmt.Println("Parsing Repositories")
	result := scanner.ParseRepos(repos, config.Verified)
	fmt.Printf("Found %d/%d valid addons\n", len(result.Addons), len(repos))

	fmt.Println("Validating Forked Verified Addons")
	for _, addon := range result.Addons {
		if !addon.Verified || !addon.Repo.Fork {
			continue
		}

		validationResult, err := scanner.ValidateForkedVerifiedAddons(*addon)
		if err != nil {
			fmt.Printf("\tFailed to validate forked verified addon: %v\n", err)
			continue
		}

		fmt.Printf("\t %s: ", addon.Repo.Id)
		switch validationResult {
		case scanner.Valid:
			fmt.Printf("Is valid\n")
			continue
		case scanner.InvalidChildTooOld:
			fmt.Printf("Repo has not been updated in 6 months -> no longer verified\n")
			addon.Verified = false
			continue
		case scanner.InvalidParentTooRecent:
			fmt.Printf("Parent repo was updated within 6 months of the fork -> no longer verified\n")
			addon.Verified = false
			continue
		}
	}

	// Count archived addons
	archivedCount := 0
	for _, addon := range result.Addons {
		if addon.Repo.Archived {
			archivedCount++
		}
	}

	// Marshal result to JSON (minified)
	jsonData, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("Failed to convert addons to JSON: %v\n", err)
		return
	}

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	// Create stats
	stats := Stats{
		LastUpdated:          time.Now().UTC().Format(time.RFC3339),
		ValidAddonsCount:     len(result.Addons),
		ArchivedAddonsCount:  archivedCount,
		InvalidAddonsCount:   len(result.InvalidAddons),
		ExecutionTimeSeconds: time.Since(startTime).Seconds(),
	}

	// Write stats.json (minified) to the same directory as output
	statsData, _ := json.Marshal(stats)
	outputDir := filepath.Dir(outputPath)
	statsPath := filepath.Join(outputDir, "stats.json")
	statsFile, _ := os.Create(statsPath)
	statsFile.Write(statsData)
	statsFile.Close()

	fmt.Printf("\nStatistics:\n")
	fmt.Printf("  Valid Addons: %d\n", stats.ValidAddonsCount)
	fmt.Printf("  Archived: %d\n", stats.ArchivedAddonsCount)
	fmt.Printf("  Invalid: %d\n", stats.InvalidAddonsCount)
	fmt.Printf("  Execution Time: %.2fs\n", stats.ExecutionTimeSeconds)

	fmt.Println("Done!")
}
