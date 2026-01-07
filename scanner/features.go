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

func detectVariable(source string, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(source)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

// removeComments strips // line comments and /* block comments */
func removeComments(source string) string {
	// remove block comments first
	blockComment := regexp.MustCompile(`(?s)/\*.*?\*/`)
	source = blockComment.ReplaceAllString(source, "")

	// remove line comments
	lineComment := regexp.MustCompile(`//.*`)
	source = lineComment.ReplaceAllString(source, "")

	return source
}

func findFeatures(fullName string, defaultBranch string, entrypoint string) (Features, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/java/%v.java",
		fullName, defaultBranch, strings.ReplaceAll(entrypoint, ".", "/"))
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return Features{}, err
	}

	source := string(bytes)
	// prevents commented out modules from being detected
	source = removeComments(source)

	// Detect Module, HUD, System, Tab variable names
	moduleVar := detectVariable(source, `(?m)\bModules\s+(\w+)\s*=\s*(Modules\.get\(\)|Systems\.get\(Modules\.class\));`)
	hudVar := detectVariable(source, `(?m)\bHud\s+(\w+)\s*=\s*(Hud\.get\(\)|Systems\.get\(Hud\.class\));`)
	systemsVar := detectVariable(source, `(?m)\bSystems\s+(\w+)\s*=\s*Systems\.get\(\);`)
	tabsVar := detectVariable(source, `(?m)\bTabs\s+(\w+)\s*=\s*Tabs\.get\(\);`)

	// recognizes added modules
	modulePattern := `(?m)(Modules\.get\(\)|Systems\.get\(Modules\.class\)|Systems\.add\(`
	if moduleVar != "" {
		modulePattern += `|` + regexp.QuoteMeta(moduleVar)
	}
	if systemsVar != "" {
		modulePattern += `|` + regexp.QuoteMeta(systemsVar)
	}
	modulePattern += `)\.add\(\s*new\s+([A-Z][A-Za-z0-9_]+)\s*\(.*?\)\s*\);?`

	// Pattern for variable-based registration: Type var = new Type(); X.add(var);
	varRegisterPattern := `(?m)([A-Z][A-Za-z0-9_]+)\s+(\w+)\s*=\s*new\s+([A-Z][A-Za-z0-9_]+)\s*\(.*?\)\s*;[\s\S]{0,200}?(Modules\.get\(\)|Commands|Systems)\.add\(\s*(\w+)\s*\)`

	// recognizes added hud elements
	hudPattern := `(?m)(Hud\.get\(\)|Systems\.get\(Hud\.class\)|Hud\.add\(`
	if hudVar != "" {
		hudPattern += `|` + regexp.QuoteMeta(hudVar)
	}
	hudPattern += `)\.register\((\w+)\.INFO\);`

	// recognizes added commands
	commandPattern := `(?m)Commands\.add\(\s*new\s+([A-Z][A-Za-z0-9_]+)\s*\(.*?\)\s*\);?`

	// recognizes added tabs
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

	var modules, hudElements, commands []Feature
	var tabs []string
	moduleSet := make(map[string]bool)
	commandSet := make(map[string]bool)
	hudSet := make(map[string]bool)
	tabSet := make(map[string]bool)

	// Extract inline new registrations for modules
	for _, match := range moduleRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 3 {
			name := splitCamelCase(match[2])
			if !moduleSet[name] {
				modules = append(modules, Feature{
					name,
					"",
				})
				moduleSet[name] = true
			}
		}
	}

	// Extract variable-based registrations
	for _, match := range varRegisterRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 6 {
			className := match[1]    // Type in declaration
			varName := match[2]      // variable name
			newClassName := match[3] // Type after 'new'
			registerType := match[4] // What's being added to (Modules/Commands/Systems)
			addedVar := match[5]     // Variable being added

			// Verify backreferences manually since RE2 doesn't support them
			if className != newClassName || varName != addedVar {
				continue
			}

			name := splitCamelCase(className)

			// Determine which category based on the add call
			if strings.Contains(registerType, "Module") || strings.Contains(registerType, "System") {
				if !moduleSet[name] {
					modules = append(modules, Feature{
						name,
						"",
					})
					moduleSet[name] = true
				}
			} else if strings.Contains(registerType, "Command") {
				if !commandSet[name] {
					commands = append(commands, Feature{
						name,
						"",
					})
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
				hudElements = append(hudElements, Feature{
					name,
					"",
				})
				hudSet[name] = true
			}
		}
	}

	// Extract commands
	for _, match := range commandRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 2 {
			name := splitCamelCase(match[1])
			if !commandSet[name] {
				commands = append(commands, Feature{
					name,
					"",
				})
				commandSet[name] = true
			}
		}
	}

	// Extract tabs (custom screens)
	for _, match := range tabRegex.FindAllStringSubmatch(source, -1) {
		if len(match) >= 3 {
			name := splitCamelCase(match[2])
			if !tabSet[name] {
				tabs = append(tabs, name)
				tabSet[name] = true
			}
		}
	}

	return Features{
		Modules:       modules,
		Commands:      commands,
		HudElements:   hudElements,
		CustomScreens: tabs,
		FeatureCount:  len(modules) + len(hudElements) + len(commands) + len(tabs),
	}, nil
}
