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

func validatePaths(verifiedAddons string, output string) error {
	if !strings.HasSuffix(verifiedAddons, ".txt") {
		return fmt.Errorf("Verified addons path must lead to a txt file")
	}

	if !strings.HasSuffix(output, ".json") {
		return fmt.Errorf("Output path must lead to a json file")
	}

	if _, err := filepath.Abs(verifiedAddons); err != nil {
		return fmt.Errorf("'%v' is not a valid path: %v", verifiedAddons, err)
	}

	if _, err := filepath.Abs(output); err != nil {
		return fmt.Errorf("'%v' is not a valid path: %v", output, err)
	}

	if _, err := os.Stat(verifiedAddons); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to read stats for '%v'", verifiedAddons)
	}

	if _, err := os.Stat(output); err == nil {
		return fmt.Errorf("Output path already exists")
	}

	return nil
}

func main() {
	args := os.Args

	if len(args) < 3 {
		fmt.Println("Not enough argument provided: verified-addons.txt output.json")
		os.Exit(1)
	}

	verifiedAddonsPath := args[1]
	outputPath := args[2]

	err := validatePaths(verifiedAddonsPath, outputPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = godotenv.Load()
	if err != nil {
		fmt.Println("Failed to load env file: ", err)
		os.Exit(1)
	}

	var key string = os.Getenv("KEY")
	scanner.InitDefaultHeaders(key)

	file, err := os.Open(verifiedAddonsPath)
	if err != nil {
		fmt.Printf("Failed to load verified addons: %v\n", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Failed to read verified addons: %v\n", err)
	}

	var verifiedAddons []string

	for line := range strings.SplitSeq(string(bytes), "\n") {
		if line != "" {
			verifiedAddons = append(verifiedAddons, strings.TrimSpace(line))
		}
	}

	fmt.Println("Locating Repositories")
	repos := scanner.Locate(verifiedAddons)
	fmt.Printf("Located %v repos\n", len(repos))
	fmt.Println("Parsing Repositories")
	addons := scanner.ParseRepos(repos)

	jsonData, err := json.Marshal(addons)
	if err != nil {
		fmt.Printf("Failed to convert addons to JSON: %v", err)
		return
	}

	file, err = os.Create(outputPath)
	if err != nil {
		fmt.Printf("Failed to create output file: %v", err)
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
