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

type config struct {
	BlacklistedRepos []string `json:"repo-blacklist"`
	BlacklistedDevs  []string `json:"developer-blacklist"`
	VerifiedAddons   []string `json:"verified"`
}

func validateConfigPath(path string) error {
	if !strings.HasSuffix(path, ".json") {
		return fmt.Errorf("Path must lead to a json file")
	}

	if _, err := filepath.Abs(path); err != nil {
		return fmt.Errorf("'%v' is not a valid path: %v", path, err)
	}

	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to read stats for '%v'", path)
	}

	return nil
}

func loadConfig(path string) (*config, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	bytes, err := io.ReadAll(fp)
	if err != nil {
		return nil, err
	}

	var cfg config

	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
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

func main() {
	args := os.Args

	if len(args) < 3 {
		fmt.Println("Not enough argument provided: config.json output.json")
		return
	}

	configPath := args[1]
	outputPath := args[2]

	err := validateConfigPath(configPath)
	if err != nil {
		fmt.Printf("Verified: %s\n", err)
		return
	}

	err = validateOutputPath(outputPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	cfg, err := loadConfig(configPath)
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

	fmt.Println("Locating Repositories")
	repos := scanner.Locate(cfg.VerifiedAddons)
	fmt.Printf("Located %v repos\n", len(repos))

	removed := 0
	for _, repo := range cfg.BlacklistedRepos {
		lower := strings.ToLower(repo)

		for fullName := range repos {
			if strings.ToLower(fullName) == lower {
				delete(repos, fullName)
				removed++
				break
			}
		}
	}

	for _, dev := range cfg.BlacklistedDevs {
		lower := strings.ToLower(dev)

		for fullName := range repos {
			if strings.HasPrefix(strings.ToLower(fullName), lower) {
				delete(repos, fullName)
				removed++
				break
			}
		}
	}

	fmt.Printf("Removed %d black listed repositories\n", removed)

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
