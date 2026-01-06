package scanner

import (
	"fmt"
	"regexp"
	"strings"
)

var moduleDescriptionRegex = regexp.MustCompile(`super\s*\(\s*[^,]+,\s*"[^"]*"\s*,\s*"([^"]*)"`)
var commandDescriptionRegex = regexp.MustCompile(`super\s*\(\s*"[^"]*"\s*,\s*"([^"]*)"`)
var hudElementDescriptionRegex = regexp.MustCompile(`new\s+HudElementInfo<[^>]*>\s*\([^,]+,\s*"[^"]*"\s*,\s*"([^"]*)"`)

func fetchDescriptions(addon *Addon) {
	baseUrl := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/java/%v", addon.Repo.Id, addon.Repo.defaultBranch, packageFromEntrypoint(addon.entrypoint))
	if len(addon.Custom.FeatureDirectories.Modules) != 0 {
		fetchModuleDescription(addon, baseUrl)
	}

	if len(addon.Custom.FeatureDirectories.Commands) != 0 {
		fetchCommandDescription(addon, baseUrl)
	}

	if len(addon.Custom.FeatureDirectories.HudElements) != 0 {
		fetchHudDescription(addon, baseUrl)
	}
}

func fetchModuleDescription(addon *Addon, baseUrl string) {
	for i := range addon.Features.Modules {
		className := strings.ReplaceAll(addon.Features.Modules[i].Name, " ", "")

		for _, directory := range addon.Custom.FeatureDirectories.Modules {
			moduleUrl := fmt.Sprintf("%s/%s/%s.java", baseUrl, directory, className)

			fileContent, err := fetchFile(moduleUrl)
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
		}
	}
}

func fetchCommandDescription(addon *Addon, baseUrl string) {
	for i := range addon.Features.Commands {
		className := strings.ReplaceAll(addon.Features.Commands[i].Name, " ", "")

		for _, directory := range addon.Custom.FeatureDirectories.Commands {
			commandUrl := fmt.Sprintf("%s/%s/%s.java", baseUrl, directory, className)

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
		}
	}
}

func fetchHudDescription(addon *Addon, baseUrl string) {
	for i := range addon.Features.HudElements {
		className := strings.ReplaceAll(addon.Features.HudElements[i].Name, " ", "")

		for _, directory := range addon.Custom.FeatureDirectories.HudElements {
			hudUrl := fmt.Sprintf("%s/%s/%s.java", baseUrl, directory, className)

			fileContent, err := fetchFile(hudUrl)
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
		}
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
