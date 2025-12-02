package scanner

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// normalizeAuthors converts various author formats to []string
func normalizeAuthors(authors interface{}) []string {
	if authors == nil {
		return []string{}
	}

	switch v := authors.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, author := range v {
			if str, ok := author.(string); ok {
				result = append(result, str)
			} else if m, ok := author.(map[string]interface{}); ok {
				// Handle object format like {"name": "AuthorName"}
				if name, ok := m["name"].(string); ok {
					result = append(result, name)
				}
			}
		}
		return result
	case string:
		// Single string author
		return []string{v}
	case []string:
		// Already correct format
		return v
	default:
		return []string{}
	}
}

// normalizeMeteorEntrypoints converts various entrypoint formats to []string
func normalizeMeteorEntrypoints(meteor interface{}) []string {
	if meteor == nil {
		return []string{}
	}

	switch v := meteor.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, entry := range v {
			if str, ok := entry.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case string:
		// Single string entrypoint
		return []string{v}
	case []string:
		// Already correct format
		return v
	case map[string]interface{}:
		// Handle map format - extract values
		result := make([]string, 0, len(v))
		for _, entry := range v {
			if str, ok := entry.(string); ok {
				result = append(result, str)
			} else if arr, ok := entry.([]interface{}); ok {
				for _, item := range arr {
					if str, ok := item.(string); ok {
						result = append(result, str)
					}
				}
			}
		}
		return result
	default:
		return []string{}
	}
}

func getRepo(fullName string) (*repository, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v", fullName)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, "", err
	}

	var repo repository

	err = json.Unmarshal(bytes, &repo)
	if err != nil {
		return nil, "", err
	}

	return &repo, string(bytes), nil
}

func getFabricModJson(fullName string, defaultBranch string) (*fabric, string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/resources/fabric.mod.json", fullName, defaultBranch)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, "", err
	}

	if string(bytes) == "404: Not Found" {
		return nil, "", fmt.Errorf("fabric.mod.json not found in expected location")
	}

	var fabricModJson fabric

	err = json.Unmarshal(bytes, &fabricModJson)
	if err != nil {
		return nil, "", fmt.Errorf("Invalid fabric.mod.json structure: %v", err)
	}

	// Normalize meteor entrypoints
	meteorEntries := normalizeMeteorEntrypoints(fabricModJson.Entrypoints.Meteor)
	if len(meteorEntries) == 0 {
		return nil, "", fmt.Errorf("No meteor entrypoint found in fabric.mod.json - not a Meteor addon")
	}

	return &fabricModJson, string(bytes), nil
}

func getCustomProperties(fullName string, defaultBranch string) (*Custom, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/meteor-addon-list.json", fullName, defaultBranch)

	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, err
	}

	var customData Custom

	if string(bytes) == "404: Not Found" {
		return &customData, nil
	}

	err = json.Unmarshal(bytes, &customData)
	if err != nil {
		return nil, err
	}

	// Validate all supported versions
	validVersions := make([]string, 0, len(customData.SupportedVersions))
	for _, v := range customData.SupportedVersions {
		v = strings.TrimSpace(v)
		if mcVersionRegex.MatchString(v) {
			validVersions = append(validVersions, v)
		}
	}

	// Sort versions from highest to lowest
	sort.Slice(validVersions, func(i, j int) bool {
		return compareMCVersions(validVersions[i], validVersions[j])
	})

	customData.SupportedVersions = validVersions

	var validTags []string

	for _, tag := range customData.Tags {
		exists, realTag := validTag(tag)

		if exists {
			validTags = append(validTags, realTag)
		}
	}

	customData.Tags = validTags

	return &customData, nil
}

func getIcon(fullName string, defaultBranch string, icon string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/resources/%v", fullName, defaultBranch, icon)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	if string(bytes) == "404: Not Found" {
		return "", nil
	}

	return url, nil
}
