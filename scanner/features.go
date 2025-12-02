package scanner

import (
	"fmt"
	"regexp"
	"strings"
)

func splitCamelCase(input string) string {
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	return re.ReplaceAllString(input, "$1 $2")
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func detectVariable(source string, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(source)
	if len(match) >= 2 {
		return match[1]
	}
	return "" // fallback to default pattern
}

func findFeatures(fullName string, defaultBranch string, entrypoint string, addonName string) (Features, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/java/%v.java",
		fullName, defaultBranch, strings.ReplaceAll(entrypoint, ".", "/"))
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return Features{}, err
	}

	source := string(bytes)

	// Detect Module, HUD, System, Tab variable names
	moduleVar := detectVariable(source, `(?m)\bModules\s+(\w+)\s*=\s*(Modules\.get\(\)|Systems\.get\(Modules\.class\));`)
	hudVar := detectVariable(source, `(?m)\bHud\s+(\w+)\s*=\s*(Hud\.get\(\)|Systems\.get\(Hud\.class\));`)
	systemsVar := detectVariable(source, `(?m)\bSystems\s+(\w+)\s*=\s*Systems\.get\(\);`)
	tabsVar := detectVariable(source, `(?m)\bTabs\s+(\w+)\s*=\s*Tabs\.get\(\);`)

	// Build flexible patterns for modules (including Systems.add)
	modulePattern := `(?m)(Modules\.get\(\)|Systems\.get\(Modules\.class\)|Systems\.add\(`
	if moduleVar != "" {
		modulePattern += `|` + regexp.QuoteMeta(moduleVar)
	}
	if systemsVar != "" {
		modulePattern += `|` + regexp.QuoteMeta(systemsVar)
	}
	modulePattern += `)\.add\(\s*new\s+([A-Z][A-Za-z0-9_]+)\s*\(.*?\)\s*\);?`

	// Pattern for variable-based registration: Type var = new Type(); X.add(var);
	// Note: Go's RE2 doesn't support backreferences, so we match the pattern and verify manually
	varRegisterPattern := `(?m)([A-Z][A-Za-z0-9_]+)\s+(\w+)\s*=\s*new\s+([A-Z][A-Za-z0-9_]+)\s*\(.*?\)\s*;[\s\S]{0,200}?(Modules\.get\(\)|Commands|Systems)\.add\(\s*(\w+)\s*\)`

	// HUD pattern (including Hud.add)
	hudPattern := `(?m)(Hud\.get\(\)|Systems\.get\(Hud\.class\)|Hud\.add\(`
	if hudVar != "" {
		hudPattern += `|` + regexp.QuoteMeta(hudVar)
	}
	hudPattern += `)\.register\((\w+)\.INFO\);`

	// Commands pattern (inline new and variable-based)
	commandPattern := `(?m)Commands\.add\(\s*new\s+([A-Z][A-Za-z0-9_]+)\s*\(.*?\)\s*\);?`

	// Tabs pattern - matches both Tabs.add() and Tabs.get().add()
	tabPattern := `(?m)(Tabs\.get\(\)\.add|Tabs\.add`
	if tabsVar != "" {
		tabPattern += `|` + regexp.QuoteMeta(tabsVar) + `\.add`
	}
	tabPattern += `)\(\s*new\s+([A-Z][A-Za-z0-9_]+)\s*\(.*?\)\s*\);?`

	moduleRegex := regexp.MustCompile(modulePattern)
	hudRegex := regexp.MustCompile(hudPattern)
	commandRegex := regexp.MustCompile(commandPattern)
	varRegisterRegex := regexp.MustCompile(varRegisterPattern)
	tabRegex := regexp.MustCompile(tabPattern)

	var modules, hudElements, commands, customScreens []string
	moduleSet := make(map[string]bool)
	commandSet := make(map[string]bool)
	hudSet := make(map[string]bool)
	customScreenSet := make(map[string]bool)

	// Extract inline new registrations for modules
	for _, match := range moduleRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 3 {
			name := splitCamelCase(match[2])
			if !moduleSet[name] {
				modules = append(modules, name)
				moduleSet[name] = true
			}
		}
	}

	// Extract variable-based registrations
	for _, match := range varRegisterRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 6 {
			className := match[1]       // Type in declaration
			varName := match[2]         // variable name
			newClassName := match[3]    // Type after 'new'
			registerType := match[4]    // What's being added to (Modules/Commands/Systems)
			addedVar := match[5]        // Variable being added

			// Verify backreferences manually since RE2 doesn't support them
			if className != newClassName || varName != addedVar {
				continue
			}

			name := splitCamelCase(className)

			// Determine which category based on the add call
			if strings.Contains(registerType, "Module") || strings.Contains(registerType, "System") {
				if !moduleSet[name] {
					modules = append(modules, name)
					moduleSet[name] = true
				}
			} else if strings.Contains(registerType, "Command") {
				if !commandSet[name] {
					commands = append(commands, name)
					commandSet[name] = true
				}
			}
		}
	}

	// Extract HUD elements
	for _, match := range hudRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 3 {
			name := splitCamelCase(match[2])
			if !hudSet[name] {
				hudElements = append(hudElements, name)
				hudSet[name] = true
			}
		}
	}

	// Extract commands
	for _, match := range commandRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 2 {
			name := splitCamelCase(match[1])
			if !commandSet[name] {
				commands = append(commands, name)
				commandSet[name] = true
			}
		}
	}

	// Extract tabs (custom screens)
	for _, match := range tabRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 3 {
			name := splitCamelCase(match[2])
			if !customScreenSet[name] {
				customScreens = append(customScreens, name)
				customScreenSet[name] = true
			}
		}
	}

	return Features{
		Modules:       modules,
		Commands:      commands,
		HudElements:   hudElements,
		CustomScreens: customScreens,
		FeatureCount:  len(modules) + len(hudElements) + len(commands) + len(customScreens),
	}, nil
}
