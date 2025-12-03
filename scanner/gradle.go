package scanner

import (
	"fmt"
	"regexp"
	"strings"
)

// Pre-compile regexp used in resolveVariable
var identifierRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// resolveVariable resolves a variable reference like ${var} or properties["var"]
func resolveVariable(versionStr string, currentVersions, allVersions map[string]string) string {
	if versionStr == "" {
		return ""
	}

	resolved := versionStr
	lookupSources := make(map[string]string)
	for k, v := range currentVersions {
		lookupSources[k] = v
	}
	for k, v := range allVersions {
		lookupSources[k] = v
	}

	// Helper to perform replacements
	replaceVar := func(matchStr, varName string) {
		if val, ok := lookupSources[varName]; ok {
			resolved = strings.Replace(resolved, matchStr, val, 1)
		}
	}

	// 1. Handle ${...} patterns
	braceRe := regexp.MustCompile(`\$\s*\{([^}]+)\}`)

	matches := braceRe.FindAllStringSubmatch(resolved, -1)
	// Iterate reversed
	for i := len(matches) - 1; i >= 0; i-- {
		fullMatch := matches[i][0]
		innerContent := strings.TrimSpace(matches[i][1])

		// Cleanup
		innerContent = strings.ReplaceAll(innerContent, ".toString()", "")
		innerContent = strings.ReplaceAll(innerContent, " as String", "")

		var varName string

		// properties["name"] or properties['name']
		propRe := regexp.MustCompile(`properties\[["']([^"'\\]+)["']\]`)
		if m := propRe.FindStringSubmatch(innerContent); m != nil {
			varName = m[1]
		} else if strings.HasPrefix(innerContent, "properties[") {
			propRe2 := regexp.MustCompile(`properties\[['"](.*?)['"]\]`)
			if m := propRe2.FindStringSubmatch(innerContent); m != nil {
				varName = m[1]
			}
		} else if strings.Contains(innerContent, "project.property") {
			propFuncRe := regexp.MustCompile(`project\.property\s*\(\s*["']([^"']+)["']\s*\)`)
			if m := propFuncRe.FindStringSubmatch(innerContent); m != nil {
				varName = m[1]
			}
		} else if strings.HasPrefix(innerContent, "project.") {
			varName = strings.TrimPrefix(innerContent, "project.")
			varName = strings.TrimSpace(varName)
		} else if identifierRe.MatchString(innerContent) {
			varName = innerContent
		}

		if varName != "" {
			replaceVar(fullMatch, varName)
		}
	}

	// 2. Handle simple $variable patterns
	simpleRe := regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)`)
	simpleMatches := simpleRe.FindAllStringSubmatch(resolved, -1)
	for i := len(simpleMatches) - 1; i >= 0; i-- {
		fullMatch := simpleMatches[i][0]
		varName := simpleMatches[i][1]
		replaceVar(fullMatch, varName)
	}

	return resolved
}

// ParseGradleVersions extracts Minecraft and Meteor versions from Gradle content
func ParseGradleVersions(content string, existingVersions map[string]string) map[string]string {
	versions := make(map[string]string)
	for k, v := range existingVersions {
		versions[k] = v
	}

	// TOML Check
	if strings.Contains(content, "[versions]") {
		mcMatch := regexp.MustCompile(`minecraft\s*=\s*["']([^"\\]+)["']`).FindStringSubmatch(content)
		if len(mcMatch) > 1 {
			versions["minecraft_version"] = strings.TrimSpace(mcMatch[1])
		}

		meteorMatch := regexp.MustCompile(`meteor\s*=\s*["']([^"\\]+)["']`).FindStringSubmatch(content)
		if len(meteorMatch) > 1 {
			versions["meteor_version"] = strings.TrimSpace(meteorMatch[1])
		}
		return versions
	}

	// Kotlin val assignments
	valRe := regexp.MustCompile(`val\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(.*)`)
	valMatches := valRe.FindAllStringSubmatch(content, -1)
	for _, match := range valMatches {
		varName := match[1]
		rawValue := strings.TrimSpace(match[2])

		quoteMatch := regexp.MustCompile(`^["\\]([^"\\]+)["\\]`).FindStringSubmatch(rawValue)
		if len(quoteMatch) > 1 {
			versions[varName] = quoteMatch[1]
		} else {
			parts := strings.Fields(rawValue)
			if len(parts) > 0 {
				versions[varName] = parts[0]
			}
		}
	}

	// --- Minecraft Version Extraction ---
	mcRe := regexp.MustCompile(`(?:minecraft_version|minecraftVersion)\s*=\s*(?:"([^"\n]*)"|'([^'\n]*)'|([^"'\s]+))`)
	mcMatch := mcRe.FindStringSubmatch(content)
	if len(mcMatch) > 0 {
		// Find the non-empty group
		var val string
		if mcMatch[1] != "" {
			val = mcMatch[1]
		} else if mcMatch[2] != "" {
			val = mcMatch[2]
		} else {
			val = mcMatch[3]
		}
		versions["minecraft_version"] = resolveVariable(strings.TrimSpace(val), versions, existingVersions)
	} else {
		mcDepRe := regexp.MustCompile(`(?i)minecraft\s*\(?\s*['"]com\.mojang:minecraft:(.*)`)
		mcDepMatch := mcDepRe.FindStringSubmatch(content)
		if len(mcDepMatch) > 1 {
			rawLine := strings.TrimSpace(mcDepMatch[1])
			lastQuoteIdx := -1
			if idx := strings.LastIndex(rawLine, "\""); idx != -1 {
				lastQuoteIdx = idx
			} else if idx := strings.LastIndex(rawLine, "'"); idx != -1 {
				lastQuoteIdx = idx
			}

			if lastQuoteIdx != -1 {
				rawVersion := rawLine[:lastQuoteIdx]
				versions["minecraft_version"] = resolveVariable(rawVersion, versions, existingVersions)
			}
		}
	}

	// --- Meteor Version Extraction ---
	meteorRe := regexp.MustCompile(`(?i)(?:meteor[_-]?version|meteorVersion)\s*=\s*(?:"([^"\n]*)"|'([^'\n]*)'|([^"'\s]+))`)
	meteorMatch := meteorRe.FindStringSubmatch(content)
	if len(meteorMatch) > 0 {
		var val string
		if meteorMatch[1] != "" {
			val = meteorMatch[1]
		} else if meteorMatch[2] != "" {
			val = meteorMatch[2]
		} else {
			val = meteorMatch[3]
		}
		versions["meteor_version"] = resolveVariable(strings.TrimSpace(val), versions, existingVersions)
	} else {
		depRe := regexp.MustCompile(`(?i)meteor-client[:](.*)`)
		depMatch := depRe.FindStringSubmatch(content)
		if len(depMatch) > 1 {
			rawLine := strings.TrimSpace(depMatch[1])
			lastQuoteIdx := -1
			if idx := strings.LastIndex(rawLine, "\""); idx != -1 {
				lastQuoteIdx = idx
			} else if idx := strings.LastIndex(rawLine, "'"); idx != -1 {
				lastQuoteIdx = idx
			}

			var rawVersionString string
			if lastQuoteIdx != -1 {
				rawVersionString = rawLine[:lastQuoteIdx]
			} else {
				parts := strings.Fields(rawLine)
				if len(parts) > 0 {
					rawVersionString = parts[0]
				}
			}
			versions["meteor_version"] = resolveVariable(rawVersionString, versions, existingVersions)
		} else {
			fileRe := regexp.MustCompile(`(?i)files\s*\((?:.*?)meteor-client-([a-zA-Z0-9\.\-_]+)\.jar(?:.*?)\)`)
			fileMatch := fileRe.FindStringSubmatch(content)
			if len(fileMatch) > 1 {
				versions["meteor_version"] = fileMatch[1]
			}
		}
	}

	// Normalization and Validation
	if v, ok := versions["meteor_version"]; ok {
		if strings.Contains(v, "stonecutter") || strings.Contains(v, "current.version") {
			versions["meteor_version"] = "SNAPSHOT"
		} else if !regexp.MustCompile(`^[a-zA-Z0-9\.\-_]+$`).MatchString(v) {
			if strings.TrimSpace(v) == "" {
				delete(versions, "meteor_version")
			}
		}
	}

	return versions
}

// parseGradleVersions fetches and parses gradle files to extract versions
func parseGradleVersions(owner, repo, defaultBranch string) (minecraftVersion, meteorVersion string) {
	// Priority order: gradle/libs.versions.toml → libs.versions.toml → gradle.properties → build.gradle → build.gradle.kts

	// Try gradle/libs.versions.toml first
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/gradle/libs.versions.toml", owner, repo, defaultBranch)
	bytes, err := MakeGetRequest(url)
	if err == nil && string(bytes) != "404: Not Found" {
		versions := ParseGradleVersions(string(bytes), nil)
		if mc, ok := versions["minecraft_version"]; ok {
			minecraftVersion = mc
		}
		if meteor, ok := versions["meteor_version"]; ok {
			meteorVersion = meteor
		}
		if minecraftVersion != "" {
			return minecraftVersion, meteorVersion
		}
	}

	// Try libs.versions.toml in root (some addons put it there)
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/libs.versions.toml", owner, repo, defaultBranch)
	bytes, err = MakeGetRequest(url)
	if err == nil && string(bytes) != "404: Not Found" {
		versions := ParseGradleVersions(string(bytes), nil)
		if mc, ok := versions["minecraft_version"]; ok {
			minecraftVersion = mc
		}
		if meteor, ok := versions["meteor_version"]; ok {
			meteorVersion = meteor
		}
		if minecraftVersion != "" {
			return minecraftVersion, meteorVersion
		}
	}

	// Try gradle.properties
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/gradle.properties", owner, repo, defaultBranch)
	bytes, err = MakeGetRequest(url)
	allVersions := make(map[string]string)
	if err == nil && string(bytes) != "404: Not Found" {
		versions := ParseGradleVersions(string(bytes), nil)
		for k, v := range versions {
			allVersions[k] = v
		}
		if mc, ok := versions["minecraft_version"]; ok {
			minecraftVersion = mc
		}
		if meteor, ok := versions["meteor_version"]; ok {
			meteorVersion = meteor
		}
		if minecraftVersion != "" {
			return minecraftVersion, meteorVersion
		}
	}

	// Try build.gradle
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/build.gradle", owner, repo, defaultBranch)
	bytes, err = MakeGetRequest(url)
	if err == nil && string(bytes) != "404: Not Found" {
		versions := ParseGradleVersions(string(bytes), allVersions)
		if mc, ok := versions["minecraft_version"]; ok {
			minecraftVersion = mc
		}
		if meteor, ok := versions["meteor_version"]; ok {
			meteorVersion = meteor
		}
		if minecraftVersion != "" {
			return minecraftVersion, meteorVersion
		}
	}

	// Try build.gradle.kts
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/build.gradle.kts", owner, repo, defaultBranch)
	bytes, err = MakeGetRequest(url)
	if err == nil && string(bytes) != "404: Not Found" {
		versions := ParseGradleVersions(string(bytes), allVersions)
		if mc, ok := versions["minecraft_version"]; ok {
			minecraftVersion = mc
		}
		if meteor, ok := versions["meteor_version"]; ok {
			meteorVersion = meteor
		}
	}

	return minecraftVersion, meteorVersion
}
