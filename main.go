package main

import (
	"dev/cqb13/meteor-addon-scanner/scanner"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

func validateInputPath(input string) error {
	if input == "--" {
		return nil
	}

	if !strings.HasSuffix(input, ".txt") {
		return fmt.Errorf("Path must lead to a txt file")
	}

	if _, err := filepath.Abs(input); err != nil {
		return fmt.Errorf("'%v' is not a valid path: %v", input, err)
	}

	if _, err := os.Stat(input); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to read stats for '%v'", input)
	}

	return nil
}

func validateOutputPath(output string) error {
	if !strings.HasSuffix(output, ".json") {
		return fmt.Errorf("Output path must lead to a json file")
	}

	if _, err := filepath.Abs(output); err != nil {
		return fmt.Errorf("'%v' is not a valid path: %v", output, err)
	}

	if _, err := os.Stat(output); err == nil {
		return fmt.Errorf("Output path already exists")
	}

	return nil
}

func loadRepoList(path string) ([]string, error) {
	if path == "--" {
		return make([]string, 0), nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load repo list: %v\n", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to read repo list: %v\n", err)
	}

	var repos []string

	for line := range strings.SplitSeq(string(bytes), "\n") {
		if line != "" {
			repos = append(repos, strings.TrimSpace(line))
		}
	}

	return repos, nil
}

func main() {
	args := os.Args

	if len(args) < 4 {
		fmt.Println("Not enough argument provided: verified-addons.txt output.json")
		return
	}

	verifiedAddonsPath := args[1]
	blackListedAddonsPath := args[2]
	outputPath := args[3]

	err := validateInputPath(verifiedAddonsPath)
	if err != nil {
		fmt.Printf("Verified: %s\n", err)
		return
	}

	err = validateInputPath(blackListedAddonsPath)
	if err != nil {
		fmt.Printf("Black-listed: %s\n", err)
		return
	}

	err = validateOutputPath(outputPath)
	if err != nil {
		fmt.Println(err)
		return
	}

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

	verifiedAddons, err := loadRepoList(verifiedAddonsPath)
	if err != nil {
		fmt.Printf("Verified: %s\n", err)
		return
	}

	blackListedAddons, err := loadRepoList(blackListedAddonsPath)
	if err != nil {
		fmt.Printf("Black-listed: %s\n", err)
		return
	}

	fmt.Println("Locating Repositories")
	repos := scanner.Locate(verifiedAddons)
	fmt.Printf("Located %v repos\n", len(repos))

	removed := 0
	for _, repo := range blackListedAddons {
		lower := strings.ToLower(repo)

		for fullName := range repos {
			if strings.ToLower(fullName) == lower {
				delete(repos, fullName)
				removed++
				break
			}
		}
	}

	fmt.Printf("Removed %d/%d black listed repositories\n", removed, len(blackListedAddons))

	fmt.Println("Parsing Repositories")
	addons := scanner.ParseRepos(repos)
	fmt.Printf("Found %d/%d valid addons\n", len(addons), len(repos))

	fmt.Println("Validating Forked Verified Addons")
	for _, addon := range addons {
		if !addon.Verified || !addon.Repo.Fork {
			continue
		}

		result, err := scanner.ValidateForkedVerifiedAddons(*addon)
		if err != nil {
			fmt.Printf("\tFailed to validate forked verified addon: %v\n", err)
			continue
		}

		fmt.Printf("\t %s: ", addon.Repo.Id)
		switch result {
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

	jsonData, err := json.Marshal(addons)
	if err != nil {
		fmt.Printf("Failed to convert addons to JSON: %v\n", err)
		return
	}

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\b", err)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("Done!")
}
