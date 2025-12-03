package scanner

import (
	"fmt"
	"strings"
	"sync"
)

// isActualTemplate determines if an addon is truly just a template with no real content
// by checking if ALL features contain "Example" (case-insensitive)
func isActualTemplate(features Features) bool {
	// If no features at all, it's a template
	if features.FeatureCount == 0 {
		return true
	}

	// Check if ALL features contain "Example" (case-insensitive)
	// If ANY feature does NOT contain "Example", it's a real addon

	// Check modules
	for _, module := range features.Modules {
		if !strings.Contains(strings.ToLower(module), "example") {
			return false
		}
	}

	// Check commands
	for _, cmd := range features.Commands {
		if !strings.Contains(strings.ToLower(cmd), "example") {
			return false
		}
	}

	// Check HUD elements
	for _, hud := range features.HudElements {
		if !strings.Contains(strings.ToLower(hud), "example") {
			return false
		}
	}

	// Check custom screens
	for _, screen := range features.CustomScreens {
		if !strings.Contains(strings.ToLower(screen), "example") {
			return false
		}
	}

	// All features contain "Example" = template
	return true
}

func findVersion(fullName string, defaultBranch string) (string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("Invalid repo name format")
	}

	minecraftVersion, _ := parseGradleVersions(parts[0], parts[1], defaultBranch)

	if minecraftVersion == "" {
		return "", fmt.Errorf("Could not find Minecraft version")
	}

	return minecraftVersion, nil
}

func ParseRepo(fullName string) (*Addon, error) {
	repo, repoStr, err := getRepo(fullName)
	if err != nil {
		return nil, err
	}

	fabricModJson, fabricStr, err := getFabricModJson(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	// Normalize entrypoints and authors
	meteorEntries := normalizeMeteorEntrypoints(fabricModJson.Entrypoints.Meteor)
	if len(meteorEntries) == 0 {
		return nil, fmt.Errorf("Missing meteor entrypoint")
	}

	// ensure a description is present
	description := repo.Description
	if description == "" {
		description = fabricModJson.Description
	}

	// find authors from fabric.mod.json or from github username
	authors := normalizeAuthors(fabricModJson.Authors)
	if len(authors) == 0 {
		authors = append(authors, repo.Owner.Login)
	}

	downloads, latestRelease, downloadCount, err := getReleaseDetails(fullName)
	if err != nil {
		return nil, err
	}

	icon, err := getIcon(fullName, repo.DefaultBranch, fabricModJson.Icon)
	if err != nil {
		return nil, err
	}

	invite, err := findDiscordServer(fullName, repo.DefaultBranch, repoStr, fabricStr)
	if err != nil {
		return nil, err
	}

	features, err := findFeatures(fullName, repo.DefaultBranch, meteorEntries[0])
	if err != nil {
		return nil, err
	}

	// Template detection - check if id is "addon-template" AND all features contain "Example"
	if fabricModJson.Id == "addon-template" && isActualTemplate(features) {
		return nil, nil // Silently skip actual templates
	}

	version, err := findVersion(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	site := repo.Homepage

	// prevent the homepage being a discord invite
	if inviteRegex.MatchString(site) {
		site = ""
	}

	customProperties, err := getCustomProperties(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	addon := Addon{
		Name:        fabricModJson.Name,
		Description: description,
		McVersion:   version,
		Authors:     authors,
		Features:    features,
		Verified:    false,
		Repo: Repo{
			Id:           fullName,
			Owner:        repo.Owner.Login,
			Name:         repo.Name,
			Archived:     repo.Archived,
			Fork:         repo.Fork,
			Stars:        repo.Stars,
			Downloads:    downloadCount,
			LastUpdate:   repo.PushedAt,
			CreationDate: repo.CreatedAt,
		},
		Links: Links{
			Github:        repo.HtmlUrl,
			Downloads:     downloads,
			LatestRelease: latestRelease,
			Discord:       invite,
			Icon:          icon,
			Homepage:      site,
		},
		Custom: *customProperties,
	}

	return &addon, nil
}

func ParseRepos(repos map[string]bool, verifiedRepos []string) *ScanResult {
	// Create verified set for O(1) lookup
	verifiedSet := make(map[string]bool)
	for _, repo := range verifiedRepos {
		verifiedSet[strings.ToLower(repo)] = true
	}

	var addons []*Addon
	var invalidAddons []InvalidAddon
	var addonsMutex sync.Mutex
	var invalidMutex sync.Mutex
	var wg sync.WaitGroup

	// Semaphore for concurrency control (10 concurrent goroutines)
	semaphore := make(chan struct{}, 10)

	for repo := range repos {
		wg.Add(1)

		go func(repoName string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore

			fmt.Printf("\tParsing: %s\n", repoName)

			addon, err := ParseRepo(repoName)
			if err != nil {
				invalidMutex.Lock()
				invalidAddons = append(invalidAddons, InvalidAddon{
					Name:   repoName,
					URL:    fmt.Sprintf("https://github.com/%s", repoName),
					Reason: err.Error(),
				})
				invalidMutex.Unlock()
				fmt.Printf("\t\tFailed to parse %s: %v\n", repoName, err)
				return
			}

			// If addon is nil (template skip), silently skip without adding to invalid
			if addon == nil {
				fmt.Printf("\t\tSkipped template: %s\n", repoName)
				return
			}

			// Set verified status
			addon.Verified = verifiedSet[strings.ToLower(repoName)]

			// Thread-safe append
			addonsMutex.Lock()
			addons = append(addons, addon)
			addonsMutex.Unlock()
		}(repo)
	}

	wg.Wait()

	return &ScanResult{
		Addons:        addons,
		InvalidAddons: invalidAddons,
	}
}
