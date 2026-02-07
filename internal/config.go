package internal

import (
	"dev/cqb13/meteor-addon-scanner/scanner"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func LoadConfig(path string) (*scanner.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file: %v", err)
	}
	defer file.Close()

	var config scanner.Config
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

func LoadInvalidRepoLog(path string, invalidRepoLog map[string]any) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return false
	}

	for line := range strings.SplitSeq(string(bytes), "\n") {
		invalidRepoLog[line] = nil
	}

	return true
}
