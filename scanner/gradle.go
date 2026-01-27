package scanner

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// fetches and parses gradle files to extract minecraft version
func getMinecraftVersion(fullName string, defaultBranch string) string {
	// Priority order: gradle/libs.versions.toml → libs.versions.toml → gradle.properties → build.gradle → build.gradle.kts

	// Try gradle/libs.versions.toml first
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/gradle/libs.versions.toml", fullName, defaultBranch)
	if mc, ok := fetchAndParseGradleFile(url); ok {
		return mc
	}

	// Try libs.versions.toml in root (some addons put it there)
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/libs.versions.toml", fullName, defaultBranch)
	if mc, ok := fetchAndParseGradleFile(url); ok {
		return mc
	}

	// Try gradle.properties
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/gradle.properties", fullName, defaultBranch)
	if mc, ok := fetchAndParseGradleFile(url); ok {
		return mc
	}

	// Try build.gradle
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/build.gradle", fullName, defaultBranch)
	if mc, ok := fetchAndParseGradleFile(url); ok {
		return mc
	}

	// Try build.gradle.kts
	url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/build.gradle.kts", fullName, defaultBranch)
	if mc, ok := fetchAndParseGradleFile(url); ok {
		return mc
	}

	return ""
}

func fetchAndParseGradleFile(url string) (string, bool) {
	bytes, err := MakeGetRequest(url)
	if err == nil && string(bytes) != "404: Not Found" {
		versions := parseGradleVersions(string(bytes))
		if mc, ok := versions["minecraft_version"]; ok {
			if mcVersionRegex.MatchString(mc) {
				return mc, true
			}
		}
	}

	return "", false
}

// resolves a variable reference like ${var} or properties["var"]
func resolveVariable(versionStr string, versions map[string]string) string {
	if versionStr == "" {
		return ""
	}

	resolved := versionStr

	replaceVar := func(matchStr, varName string) {
		if val, ok := versions[varName]; ok {
			resolved = strings.Replace(resolved, matchStr, val, 1)
		}
	}

	// Handle ${...} patterns
	braceRe := regexp.MustCompile(`\$\s*\{([^}]+)\}`)

	matches := braceRe.FindAllStringSubmatch(resolved, -1)

	for i := len(matches) - 1; i >= 0; i-- {
		fullMatch := matches[i][0]
		innerContent := strings.TrimSpace(matches[i][1])

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
		} else if after, ok := strings.CutPrefix(innerContent, "project."); ok {
			varName = after
			varName = strings.TrimSpace(varName)
		} else if identifierRe.MatchString(innerContent) {
			varName = innerContent
		}

		if varName != "" {
			replaceVar(fullMatch, varName)
		}
	}

	// Handle $variable patterns
	simpleRe := regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)`)
	simpleMatches := simpleRe.FindAllStringSubmatch(resolved, -1)
	for i := len(simpleMatches) - 1; i >= 0; i-- {
		fullMatch := simpleMatches[i][0]
		varName := simpleMatches[i][1]
		replaceVar(fullMatch, varName)
	}

	return resolved
}

// extracts Minecraft versions from Gradle content
func parseGradleVersions(content string) map[string]string {
	versions := make(map[string]string)

	// TOML Check
	if strings.Contains(content, "[versions]") {
		mcMatch := regexp.MustCompile(`minecraft\s*=\s*["']([^"\\]+)["']`).FindStringSubmatch(content)
		if len(mcMatch) > 1 {
			versions["minecraft_version"] = strings.TrimSpace(mcMatch[1])
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

	// Minecraft Version Extraction
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
		versions["minecraft_version"] = resolveVariable(strings.TrimSpace(val), versions)
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
				versions["minecraft_version"] = resolveVariable(rawVersion, versions)
			}
		}
	}

	return versions
}
