package main

import (
	"dev/cqb13/meteor-addon-scanner/internal"
	"dev/cqb13/meteor-addon-scanner/scanner"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	args := os.Args

	if len(args) < 3 {
		fmt.Println("Not enough argument provided: config.json output.json")
		return
	}

	configPath := args[1]
	outputPath := args[2]

	err := validateOutputPath(outputPath)
	if err != nil {
		fmt.Println(err)
		return
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

	addon, err := scanner.ParseRepo("cqb13/numby-hack", config)
	if err != nil {
		fmt.Println(err)
		return
	}

	_ = addon
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
