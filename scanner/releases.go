package scanner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// isValidJarAsset checks if an asset name is a valid distributable JAR.
// Filters out development, source, documentation, and fat JARs.
func isValidJarAsset(assetName string) bool {
	name := strings.ToLower(assetName)

	if !strings.HasSuffix(name, ".jar") {
		return false
	}

	// Exclude non-distributable JAR types
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

// extractMCVersionFromFilename attempts to extract a Minecraft version from a JAR filename.
// Returns the version string (e.g., "1.21.10") or empty string if not found.
func extractMCVersionFromFilename(filename string) string {
	// Match patterns like: 1.21, 1.21.1, 1.21.10, etc.
	re := regexp.MustCompile(`(?i)(?:^|[_\-\.])(?:mc)?[_\-\.]?(1\.\d+(?:\.\d+)?)(?:[_\-\.]|\.jar$)`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// compareMCVersions compares two Minecraft version strings.
// Returns true if v1 > v2 (v1 is newer), false otherwise.
// Handles versions like "1.21", "1.21.1", "1.21.10"
func compareMCVersions(v1, v2 string) bool {
	if v1 == "" {
		return false
	}
	if v2 == "" {
		return true
	}

	// Parse version parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part numerically
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int

		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &p1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &p2)
		}

		if p1 > p2 {
			return true
		}
		if p1 < p2 {
			return false
		}
	}

	return false // versions are equal
}

// shouldProcessRelease determines if a release should be processed.
// Draft releases are always skipped. If allowPrerelease is false,
// only stable releases are processed.
func shouldProcessRelease(rel release, allowPrerelease bool) bool {
	// Always skip draft releases
	if rel.Draft {
		return false
	}

	// If prereleases allowed, accept any non-draft
	if allowPrerelease {
		return true
	}

	// Otherwise, only accept stable releases
	return !rel.Prerelease
}

// findReleaseJars searches GitHub releases for valid JAR downloads.
// Paginates through all releases, filtering by allowPrerelease parameter.
// Draft releases are always skipped.
func findReleaseJars(fullName string, allowPrerelease bool) ([]string, int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/releases?per_page=100&page=", fullName)

	var totalDownloadCount int = 0
	downloads := make([]string, 0)
	foundValidRelease := false
	page := 1

	for {
		bytes, err := MakeGetRequest(fmt.Sprintf("%v%v", url, page))
		if err != nil {
			return nil, 0, err
		}

		var releases []release
		err = json.Unmarshal(bytes, &releases)
		if err != nil {
			return nil, 0, err
		}

		if len(releases) == 0 {
			break
		}

		for _, rel := range releases {
			// Skip based on draft/prerelease status
			if !shouldProcessRelease(rel, allowPrerelease) {
				continue
			}

			hasValidJar := false
			for _, asset := range rel.Assets {
				// Check if valid distributable JAR
				if isValidJarAsset(asset.Name) {
					// Count downloads only from valid distributable JARs
					totalDownloadCount += asset.Downloads

					if !foundValidRelease {
						downloads = append(downloads, asset.Url)
						hasValidJar = true
					}
				}
			}

			if hasValidJar {
				foundValidRelease = true
			}
		}

		page++
	}

	return downloads, totalDownloadCount, nil
}

// getReleaseDetails fetches GitHub releases and returns download URLs.
//
// Strategy:
//   1. Fetch both stable and prerelease downloads
//   2. Determine which is the absolute latest based on GitHub's ordering
//   3. Returns: all download URLs, latest URL, total download count, error
//
// Returns: all downloads, latest release URL, total download count, error
func getReleaseDetails(fullName string) ([]string, string, int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/releases?per_page=100&page=1", fullName)

	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, "", 0, err
	}

	var releases []release
	err = json.Unmarshal(bytes, &releases)
	if err != nil {
		return nil, "", 0, err
	}

	var stableDownloads []string
	var prereleaseDownloads []string
	var latestDownload string
	var latestVersion string
	var totalDownloadCount int
	foundStable := false
	foundPrerelease := false

	// GitHub returns releases in order from newest to oldest
	for _, rel := range releases {
		// Skip drafts
		if rel.Draft {
			continue
		}

		isStable := !rel.Prerelease

		// Count downloads from all valid JARs
		for _, asset := range rel.Assets {
			if isValidJarAsset(asset.Name) {
				totalDownloadCount += asset.Downloads
			}
		}

		// Get stable release (if not found yet)
		if isStable && !foundStable {
			// Collect ALL valid JARs from this stable release
			for _, asset := range rel.Assets {
				if isValidJarAsset(asset.Name) {
					stableDownloads = append(stableDownloads, asset.Url)

					// Extract MC version from filename and track highest version
					mcVersion := extractMCVersionFromFilename(asset.Name)
					if compareMCVersions(mcVersion, latestVersion) {
						latestVersion = mcVersion
						latestDownload = asset.Url
					} else if latestDownload == "" {
						// Fallback: if no version detected, use first JAR
						latestDownload = asset.Url
					}
				}
			}
			if len(stableDownloads) > 0 {
				foundStable = true
			}
		}

		// Get prerelease (if not found yet)
		if !isStable && !foundPrerelease {
			// Collect ALL valid JARs from this prerelease
			for _, asset := range rel.Assets {
				if isValidJarAsset(asset.Name) {
					prereleaseDownloads = append(prereleaseDownloads, asset.Url)

					// Extract MC version from filename and track highest version
					mcVersion := extractMCVersionFromFilename(asset.Name)
					if compareMCVersions(mcVersion, latestVersion) {
						latestVersion = mcVersion
						latestDownload = asset.Url
					} else if latestDownload == "" {
						// Fallback: if no version detected, use first JAR
						latestDownload = asset.Url
					}
				}
			}
			if len(prereleaseDownloads) > 0 {
				foundPrerelease = true
			}
		}

		// Stop searching once we've found both
		if foundStable && foundPrerelease {
			break
		}
	}

	// Combine downloads: stable first, then prerelease
	allDownloads := append(stableDownloads, prereleaseDownloads...)

	// If no releases found at all, return empty
	if len(allDownloads) == 0 {
		return []string{}, "", totalDownloadCount, nil
	}

	return allDownloads, latestDownload, totalDownloadCount, nil
}
