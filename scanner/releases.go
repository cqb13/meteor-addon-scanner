package scanner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// filters out development, source, documentation, and fat JARs.
func isValidJarAsset(assetName string) bool {
	name := strings.ToLower(assetName)

	if !strings.HasSuffix(name, ".jar") {
		return false
	}

	excludedSuffixes := []string{
		"-dev.jar",
		"-sources.jar",
		"-all.jar",
		"-javadoc.jar",
	}

	for _, suffix := range excludedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}

	return true
}

func extractMCVersionFromFilename(filename string) string {
	// Match patterns like: 1.21, 1.21.1, 1.21.10, etc.
	re := regexp.MustCompile(`(?i)(?:^|[_\-\.])(?:mc)?[_\-\.]?(1\.\d+(?:\.\d+)?)(?:[_\-\.]|\.jar$)`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func getReleaseDetails(fullName string) ([]string, string, int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/releases?per_page=100&page=", fullName)

	var stableDownloads []string
	var prereleaseDownloads []string
	var latestDownload string
	var latestVersion string
	var totalDownloadCount int
	foundStable := false
	foundPrerelease := false
	page := 1

	for {
		bytes, err := MakeGetRequest(fmt.Sprintf("%v%v", url, page))
		if err != nil {
			return nil, "", 0, err
		}

		var releases []release
		err = json.Unmarshal(bytes, &releases)
		if err != nil {
			return nil, "", 0, err
		}

		if len(releases) == 0 {
			break
		}

		for _, rel := range releases {
			if rel.Draft {
				continue
			}

			isStable := !rel.Prerelease

			for _, asset := range rel.Assets {
				if isValidJarAsset(asset.Name) {
					totalDownloadCount += asset.Downloads
				}
			}

			if isStable && !foundStable {
				for _, asset := range rel.Assets {
					if !isValidJarAsset(asset.Name) {
						continue
					}

					stableDownloads = append(stableDownloads, asset.Url)

					// Extract MC version from filename and track highest version
					mcVersion := extractMCVersionFromFilename(asset.Name)

					if latestVersion == "" || CompareMinecraftVersions(mcVersion, latestVersion) > 0 {
						latestVersion = mcVersion
						latestDownload = asset.Url
					} else if latestDownload == "" {
						// If no version detected yet, use the first JAR as a fallback
						latestDownload = asset.Url
					}

				}
				if len(stableDownloads) > 0 {
					foundStable = true
				}
			}

			if !isStable && !foundPrerelease {
				for _, asset := range rel.Assets {
					if !isValidJarAsset(asset.Name) {
						continue
					}

					prereleaseDownloads = append(prereleaseDownloads, asset.Url)

					// Extract MC version from filename and track highest version
					mcVersion := extractMCVersionFromFilename(asset.Name)
					if latestVersion == "" || CompareMinecraftVersions(mcVersion, latestVersion) > 0 {
						latestVersion = mcVersion
						latestDownload = asset.Url
					} else if latestDownload == "" {
						// if no version detected, use first JAR
						latestDownload = asset.Url
					}
				}
				if len(prereleaseDownloads) > 0 {
					foundPrerelease = true
				}
			}

		}

		page++
	}

	allDownloads := append(stableDownloads, prereleaseDownloads...)

	if len(allDownloads) == 0 {
		return []string{}, "", totalDownloadCount, nil
	}

	return allDownloads, latestDownload, totalDownloadCount, nil
}
