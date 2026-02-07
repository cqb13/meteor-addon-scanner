package main

import (
	"dev/cqb13/meteor-addon-scanner/internal"
	"dev/cqb13/meteor-addon-scanner/scanner"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	startTime := time.Now()
	args := os.Args

	if len(args) < 3 {
		fmt.Println("Not enough argument provided: config.json output.json [invalid.txt]")
		return
	}

	configPath := args[1]
	outputPath := args[2]

	err := validateOutputPath(outputPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	invalidRepoLogPath := ""
	invalidRepoLog := make(map[string]any)
	if len(args) >= 4 {
		invalidRepoLogPath = args[3]

		ok := internal.LoadInvalidRepoLog(invalidRepoLogPath, invalidRepoLog)
		if !ok {
			fmt.Println("Failed to load invalid repo log")
		} else {
			fmt.Println("Loaded invalid repo log")
		}
	}

	err = internal.ValidateConfigPath(configPath)
	if err != nil {
		fmt.Printf("Verified: %s\n", err)
		return
	}

	config, err := internal.LoadConfig("config.json")
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
	repos := scanner.Locate(config.VerifiedAddons.Verified)
	fmt.Printf("Located %v repos\n", len(repos))

	removed := internal.RemoveBlacklistedRepositories(config, repos)
	fmt.Printf("Removed %d/%d repo blacklisted repositories\n", removed, len(config.BlacklistedRepos))

	removed = internal.RemoveBlacklistedDevelopers(config, repos)
	fmt.Printf("Removed %d repositories from blacklisted developers\n", removed)

	fmt.Println("Parsing Repositories")
	addons := scanner.ParseRepos(repos, config, invalidRepoLog)
	fmt.Printf("Found %d/%d valid addons\n", len(addons), len(repos))

	if config.VerifiedAddons.ValidateForks {
		fmt.Println("Validating forked verified addons")
		log := internal.ValidateForkedVerifiedAddons(addons)
		for addon, status := range log {
			fmt.Printf("\t%s: %s\n", addon, status)
		}
	}

	if config.VerifiedAddons.MinMinecraftVersion != "" {
		fmt.Println("Ensuring verified addons comply with minimum version")
		unverifiedAddons := internal.ValidateVerifiedAddonVersions(addons, config.VerifiedAddons.MinMinecraftVersion)
		for addon, version := range unverifiedAddons {
			fmt.Printf("\t%s: Supports %s which is bellow the required %s version -> no longer verified\n", addon, version, config.VerifiedAddons.MinMinecraftVersion)
		}
	}

	fmt.Println("Checking for suspicious addons")
	suspicious := internal.DetectSuspiciousAddons(addons, config)
	if len(suspicious) == 0 {
		fmt.Println("Found no suspicious addons")
	}

	for repo, reasons := range suspicious {
		fmt.Printf("\t%s: %s\n", repo, strings.Join(reasons, ", "))
	}

	// update invalid repo log, if used
	if invalidRepoLogPath != "" {
		file, err := os.Create(invalidRepoLogPath)
		if err != nil {
			fmt.Printf("Failed to create invalid repo log: %s\n", err)
		} else {
			for repo := range invalidRepoLog {
				fmt.Fprintf(file, "%s\n", repo)
			}

			fmt.Printf("Updated invalid repo log\n")

		}
	}

	// save addons
	jsonData, err := json.Marshal(addons)
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

	// get stats
	archivedCount := 0
	for _, addon := range addons {
		if addon.Repo.Archived {
			archivedCount++
		}
	}

	executionTime := time.Since(startTime).Seconds()
	fmt.Printf("Statistics:\n")
	fmt.Printf("  Valid Addons: %d\n", len(addons))
	fmt.Printf("  Archived: %d\n", archivedCount)
	fmt.Printf("  Invalid: %d\n", len(repos)-len(addons))
	minutes := int(executionTime) / 60
	seconds := int(executionTime) % 60
	fmt.Printf("  Execution Time: %d.%02d\n", minutes, seconds)

	fmt.Println("Done!")
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
