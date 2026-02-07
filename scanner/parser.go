package scanner

import (
	"fmt"
	"strings"
	"sync"
)

// determines if an addon is truly just a template with no real content
// by checking if ALL features contain "Example" (case-insensitive)
func isActualTemplate(features Features) bool {
	if features.FeatureCount == 0 {
		return true
	}

	for _, module := range features.Modules {
		if !strings.Contains(strings.ToLower(module.Name), "example") {
			return false
		}
	}

	for _, cmd := range features.Commands {
		if !strings.Contains(strings.ToLower(cmd.Name), "example") {
			return false
		}
	}

	for _, hud := range features.HudElements {
		if !strings.Contains(strings.ToLower(hud.Name), "example") {
			return false
		}
	}

	return true
}

func findVersion(fullName string, defaultBranch string) (string, error) {
	minecraftVersion := getMinecraftVersion(fullName, defaultBranch)

	if minecraftVersion == "" {
		return "", fmt.Errorf("Could not find Minecraft version")
	}

	return minecraftVersion, nil
}

func ParseRepo(fullName string, config *Config) (*Addon, error) {
	repo, repoStr, err := getRepo(fullName)
	if err != nil {
		return nil, err
	}

	fabricModJson, fabricStr, err := getFabricModJson(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	meteorEntries := normalizeMeteorEntrypoints(fabricModJson.Entrypoints.Meteor)
	if len(meteorEntries) == 0 {
		return nil, fmt.Errorf("No meteor entrypoint found in fabric.mod.json")
	}

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

	// Template detection, but allow actual addon template
	if fabricModJson.Id == "addon-template" && isActualTemplate(features) && strings.ToLower(repo.FullName) != "meteordevelopment/meteor-addon-template" {
		return nil, nil
	}

	version := getMinecraftVersion(fullName, repo.DefaultBranch)

	site := repo.Homepage

	// prevent the homepage being a discord invite
	if inviteRegex.MatchString(site) {
		site = ""
	}

	customProperties, err := getCustomProperties(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	if version == "" && len(customProperties.SupportedVersions) == 0 && config.RequireMinecraftVersion {
		return nil, fmt.Errorf("Could not find Minecraft version")
	}

	addon := Addon{
		Name:        fabricModJson.Name,
		Description: description,
		McVersion:   version,
		Authors:     authors,
		Features:    features,
		Verified:    false,
		entrypoint:  strings.ReplaceAll(meteorEntries[0], ".", "/"),
		Repo: Repo{
			Id:            fullName,
			defaultBranch: repo.DefaultBranch,
			Owner:         repo.Owner.Login,
			Name:          repo.Name,
			Archived:      repo.Archived,
			Fork:          repo.Fork,
			Forks:         repo.Forks,
			Stars:         repo.Stars,
			Downloads:     downloadCount,
			LastUpdate:    repo.PushedAt,
			CreationDate:  repo.CreatedAt,
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

func ParseRepos(repos map[string]bool, config *Config, invalidAddonsLog map[string]any) []*Addon {
	verifiedSet := make(map[string]bool)
	for _, repo := range config.VerifiedAddons.Verified {
		verifiedSet[strings.ToLower(repo)] = true
	}

	var addons []*Addon
	var addonsMutex sync.Mutex

	var invalidAddonsLogMutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 10)

	for repo := range repos {
		wg.Add(1)

		go func(repoName string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			_, ok := invalidAddonsLog[repoName]
			if ok {
				fmt.Printf("\tSkipping %s: Marked as invalid\n", repoName)
				return
			}

			addon, err := ParseRepo(repoName, config)
			if err != nil {
				invalidAddonsLogMutex.Lock()
				invalidAddonsLog[repoName] = nil
				invalidAddonsLogMutex.Unlock()

				fmt.Printf("\tFailed to parse %s: %v\n", repoName, err)
				return
			}

			if addon == nil {
				fmt.Printf("\tSkipped template: %s\n", repoName)
				return
			}

			addon.Verified = verifiedSet[strings.ToLower(repoName)]

			if config.ModuleDescriptions.Fetch && (config.ModuleDescriptions.OnlyVerified && addon.Verified || !config.ModuleDescriptions.OnlyVerified) && addon.Repo.Stars >= config.ModuleDescriptions.MinStarCount {
				fetchDescriptions(addon)
			}

			addonsMutex.Lock()
			addons = append(addons, addon)
			addonsMutex.Unlock()
		}(repo)
	}

	wg.Wait()

	return addons
}
