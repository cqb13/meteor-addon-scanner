package scanner

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var moduleDescriptionRegex = regexp.MustCompile(`super\s*\(\s*[^,]+,\s*"[^"]*"\s*,\s*"([^"]*)"`)
var commandDescriptionRegex = regexp.MustCompile(`super\s*\(\s*"[^"]*"\s*,\s*"([^"]*)"`)
var hudElementDescriptionRegex = regexp.MustCompile(`new\s+HudElementInfo<[^>]*>\s*\([^,]+,\s*"[^"]*"\s*,\s*"([^"]*)"`)

type treeResponse struct {
	SHA  string `json:"sha"`
	URL  string `json:"url"`
	Tree []struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
		Size int64  `json:"size,omitempty"`
		URL  string `json:"url"`
	} `json:"tree"`
	Truncated bool `json:"truncated"`
}

type FeatureType int

const (
	Module FeatureType = iota
	Command
	HudElement
)

func (f FeatureType) String() string {
	switch f {
	case Module:
		return "Module"
	case Command:
		return "Command"
	case HudElement:
		return "HudElement"
	default:
		return "Module"
	}
}

func (f FeatureType) MatchRegex() *regexp.Regexp {
	switch f {
	case Module:
		return moduleDescriptionRegex
	case Command:
		return commandDescriptionRegex
	case HudElement:
		return hudElementDescriptionRegex
	default:
		return moduleDescriptionRegex
	}
}

func fetchDescriptions(addon *Addon) {
	entryPoint := packageFromEntrypoint(addon.entrypoint)
	baseUrl := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/java/%v", addon.Repo.Id, addon.Repo.defaultBranch, entryPoint)

	searchUrl, err := MakeGetRequest(fmt.Sprintf("https://api.github.com/repos/%v/git/trees/%v?recursive=1", addon.Repo.Id, addon.Repo.defaultBranch))
	if err != nil {
		return
	}

	var response treeResponse
	if err := json.Unmarshal(searchUrl, &response); err != nil {
		fmt.Printf("\tFailed to parse %s: %v\n", addon.Name, err)
		return
	}

	if response.Truncated {
		fmt.Printf("\tWarning: %v tree was truncated by github", addon.Repo.Id)
	}

	featureClasses := make(map[string]string)
	path := fmt.Sprintf("src/main/java/%v/", entryPoint)

	for _, item := range response.Tree {
		if item.Type == "blob" && strings.HasPrefix(item.Path, path) && strings.HasSuffix(item.Path, ".java") {
			className := strings.TrimSuffix(filepath.Base(item.Path), ".java")
			_, relativePath, _ := strings.Cut(item.Path, path)
			featureClasses[className] = relativePath
		}
	}

	if len(addon.Features.Modules) != 0 {
		fetchFeatureDescription(Module, addon.Features.Modules, baseUrl, featureClasses)
	}

	if len(addon.Features.Commands) != 0 {
		fetchFeatureDescription(Command, addon.Features.Commands, baseUrl, featureClasses)
	}

	if len(addon.Features.HudElements) != 0 {
		fetchFeatureDescription(HudElement, addon.Features.HudElements, baseUrl, featureClasses)
	}
}

func fetchFeatureDescription(featureType FeatureType, features []Feature, baseUrl string, featureClasses map[string]string) {
	matchExp := featureType.MatchRegex()

	for i, feature := range features {
		className := strings.ReplaceAll(feature.Name, " ", "")

		if _, exists := featureClasses[className]; !exists {
			continue
		}

		fileContent, err := fetchFile(fmt.Sprintf("%s/%s", baseUrl, featureClasses[className]))
		if err != nil {
			continue
		}

		if !strings.Contains(fileContent, fmt.Sprintf("extends %s", featureType.String())) || !strings.Contains(fileContent, fmt.Sprintf("public %s", className)) {
			continue
		}

		matches := matchExp.FindStringSubmatch(fileContent)
		if len(matches) > 1 {
			features[i].Description = matches[1]
		}

		delete(featureClasses, className)
	}
}

func fetchFile(url string) (string, error) {
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	if string(bytes) == "404: Not Found" {
		return "", fmt.Errorf("file not found")
	}

	return string(bytes), nil
}

func packageFromEntrypoint(entrypoint string) string {
	lastDot := strings.LastIndex(entrypoint, "/")
	if lastDot == -1 {
		return entrypoint
	}

	return entrypoint[:lastDot]
}
