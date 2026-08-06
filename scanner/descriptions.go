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

func fetchDescriptions(addon *Addon) {
	entryPoint := fmt.Sprintf(packageFromEntrypoint(addon.entrypoint))
	baseUrl := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/java/%v", addon.Repo.Id, addon.Repo.defaultBranch, entryPoint)

	searchUrl, err := MakeGetRequest(fmt.Sprintf("https://api.github.com/repos/%v/git/trees/%v?recursive=1", addon.Repo.Id, addon.Repo.defaultBranch))
	if err != nil {
		return
	}

	var response TreeResponse
	if err := json.Unmarshal(searchUrl, &response); err != nil {
		fmt.Printf("\tFailed to parse %s: %v\n", addon.Name, err)
		return
	}

	if response.Truncated {
		fmt.Printf("\tWarning: %v tree was truncated by github", addon.Repo.Id)
	}

	PossibleFeatures := make(map[string]string)
	path := fmt.Sprintf("src/main/java/%v", entryPoint)

	for _, item := range response.Tree {
		if item.Type == "blob" && strings.HasPrefix(item.Path, path+"/") && strings.HasSuffix(item.Path, ".java") {
			className := strings.TrimSuffix(filepath.Base(item.Path), ".java")
			_, relativePath, _ := strings.Cut(item.Path, entryPoint)

			PossibleFeatures[className] = relativePath
		}
	}

	if len(addon.Features.Modules) != 0 {
		fetchModuleDescription(addon, baseUrl, PossibleFeatures)
	}

	if len(addon.Features.Commands) != 0 {
		fetchCommandDescription(addon, baseUrl, PossibleFeatures)
	}

	if len(addon.Features.HudElements) != 0 {
		fetchHudDescription(addon, baseUrl, PossibleFeatures)
	}
}

func fetchModuleDescription(addon *Addon, baseUrl string, PossibleFeaturesSet map[string]string) {
	for i := range addon.Features.Modules {
		className := strings.ReplaceAll(addon.Features.Modules[i].Name, " ", "")

		if _, exists := PossibleFeaturesSet[className]; !exists {
			continue
		}

		commandUrl := fmt.Sprintf("%s%s", baseUrl, PossibleFeaturesSet[className])
		fileContent, err := fetchFile(commandUrl)
		if err != nil {
			continue
		}

		if !strings.Contains(fileContent, "extends Module") || !strings.Contains(fileContent, fmt.Sprintf("public %s", className)) {
			continue
		}

		matches := moduleDescriptionRegex.FindStringSubmatch(fileContent)
		desc := ""
		if len(matches) > 1 {
			desc = matches[1]
		}

		addon.Features.Modules[i].Description = desc
		delete(PossibleFeaturesSet, className)
	}
}

func fetchCommandDescription(addon *Addon, baseUrl string, PossibleFeaturesSet map[string]string) {
	for i := range addon.Features.Commands {
		className := strings.ReplaceAll(addon.Features.Commands[i].Name, " ", "")

		if _, exists := PossibleFeaturesSet[className]; !exists {
			continue
		}

		commandUrl := fmt.Sprintf("%s%s", baseUrl, PossibleFeaturesSet[className])
		fileContent, err := fetchFile(commandUrl)
		if err != nil {
			continue
		}

		if !strings.Contains(fileContent, "extends Command") || !strings.Contains(fileContent, fmt.Sprintf("public %s", className)) {
			continue
		}
		matches := commandDescriptionRegex.FindStringSubmatch(fileContent)
		desc := ""
		if len(matches) > 1 {
			desc = matches[1]
		}

		addon.Features.Commands[i].Description = desc
		delete(PossibleFeaturesSet, className)
	}
}

func fetchHudDescription(addon *Addon, baseUrl string, PossibleFeaturesSet map[string]string) {
	for i := range addon.Features.HudElements {
		className := strings.ReplaceAll(addon.Features.HudElements[i].Name, " ", "")

		if _, exists := PossibleFeaturesSet[className]; !exists {
			continue
		}

		commandUrl := fmt.Sprintf("%s%s", baseUrl, PossibleFeaturesSet[className])
		fileContent, err := fetchFile(commandUrl)
		if err != nil {
			continue
		}

		if !strings.Contains(fileContent, "extends HudElement") || !strings.Contains(fileContent, fmt.Sprintf("public %s", className)) {
			continue
		}

		matches := hudElementDescriptionRegex.FindStringSubmatch(fileContent)
		desc := ""
		if len(matches) > 1 {
			desc = matches[1]
		}

		addon.Features.HudElements[i].Description = desc
		delete(PossibleFeaturesSet, className)
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
