package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	LimitFeatures bool
	FeatureLimit  int
	RetryCount    int
	ReposPerPage  int
}

func ParseConfig() Config {
	fmt.Println("Loading config file")
	var config Config

	filename := "scanner.config"

	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Println("\tFailed to make create absolute path: ", err)
		os.Exit(1)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Println("\tFailed to read file: ", err)
		os.Exit(1)
	}

	str := string(content)

	parts := strings.Split(str, "\n")

	for i := range parts {
		if strings.HasPrefix(parts[i], "LIMIT_FEATURES=") {
			strVal := strings.Replace(parts[i], "LIMIT_FEATURES=", "", 1)
			boolVal, err := strconv.ParseBool(strVal)
			if err != nil {
				fmt.Println("\tFailed to parse LIMIT_FEATURES to boolean")
				os.Exit(1)
			}
			fmt.Printf("\tLimit Features = %v\n", boolVal)
			config.LimitFeatures = boolVal
		}

		if strings.HasPrefix(parts[i], "FEATURE_LIMIT=") {
			strVal := strings.Replace(parts[i], "FEATURE_LIMIT=", "", 1)
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				fmt.Println("\tFailed to parse FEATURE_LIMIT to int")
				os.Exit(1)
			}
			fmt.Printf("\tFeature Limit = %v\n", intVal)
			config.FeatureLimit = intVal
		}

		if strings.HasPrefix(parts[i], "RETRY_COUNT=") {
			strVal := strings.Replace(parts[i], "RETRY_COUNT=", "", 1)
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				fmt.Println("\tFailed to parse RETRY_COUNT to int")
				os.Exit(1)
			}
			fmt.Printf("\tRetry Count = %v\n", intVal)
			config.RetryCount = intVal
		}

		if strings.HasPrefix(parts[i], "REPOS_PER_PAGE=") {
			strVal := strings.Replace(parts[i], "REPOS_PER_PAGE=", "", 1)
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				fmt.Println("\tFailed to parse REPOS_PER_PAGE to int")
				os.Exit(1)
			}
			fmt.Printf("\tRepos Per Page = %v\n", intVal)
			config.ReposPerPage = intVal
		}
	}

	if config.FeatureLimit <= 0 {
		fmt.Println("\tFeature limit must be greater than 0")
		os.Exit(1)
	}

	if config.RetryCount <= 0 {
		fmt.Println("\tRetry count must be greater than 0")
		os.Exit(1)
	}

	if config.ReposPerPage <= 0 {
		fmt.Println("\tRepos per Page must be greater than 0")
		os.Exit(1)
	}

	return config
}
