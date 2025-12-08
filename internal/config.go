package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	BlacklistedRepos        []string `json:"repo-blacklist"`
	BlacklistedDevs         []string `json:"developer-blacklist"`
	Verified                []string `json:"verified"`
	MinimumMinecraftVersion *string  `json:"minimum_minecraft_version"`
	IgnoreArchived          bool     `json:"ignore_archived"`
	IgnoreForks             bool     `json:"ignore_forks"`
	SuspicionTriggers       struct {
		NameLength        int `json:"name_len"`
		DescriptionLength int `json:"description_len"`
		FeatureCount      int `json:"feature_count"`
		SupportedVersions int `json:"supported_versions"`
	} `json:"suspicion_triggers"`
}

func LoadConfig(path string) (*Config, error) {
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

func ValidateConfigPath(path string) error {
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
