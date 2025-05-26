package main

import (
	"dev/cqb13/meteor-addon-scanner/scanner"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func validatePaths(verifiedAddons string, output string) error {
	if !fs.ValidPath(verifiedAddons) {
		return fmt.Errorf("'%v' is not a valid path", verifiedAddons)
	}

	if !fs.ValidPath(output) {
		return fmt.Errorf("'%v' is not a valid path", output)
	}

	if !strings.HasSuffix(verifiedAddons, ".json") {
		return fmt.Errorf("Verified addons path must lead to a json file")
	}

	if !strings.HasSuffix(output, ".json") {
		return fmt.Errorf("Output path must lead to a json file")
	}

	_, err := os.Stat(verifiedAddons)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to read stats for '%v'", verifiedAddons)
	}

	_, err = os.Stat(output)
	if err == nil {
		return fmt.Errorf("Output path already exists")
	}

	return nil
}

func main() {
	args := os.Args

	if len(args) < 3 {
		fmt.Println("Not enough argument provided: verified-addons.json output.json")
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

	// fmt.Println("Locating Repositories")
	// repos := scanner.Locate()
	// fmt.Printf("Located %v repos\n", len(repos))
	fmt.Println("Parsing Repositories")
	var repos = [...]string{"cqb13/Numby-hack"}
	scanner.ParseRepos(repos)
}
