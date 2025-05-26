package scanner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Addon struct {
	Name         string
	Description  string
	McVersion    string
	Authors      []string
	Features     []string
	FeatureCount int
	Verified     bool
	Repo         Repo
	Links        Links
}

type Repo struct {
	Id         string
	Owner      string
	Name       string
	Archived   bool
	Fork       bool
	Stars      int
	Downloads  int
	LastUpdate string
}

type Links struct {
	Github   string
	Download string
	Discord  string
	Homepage string
}

// matches patterns like `add(new SomeFeatureName(...))`
// and captures the feature name (e.g., "SomeFeatureName")
var FEATURE_RE = regexp.MustCompile(`(?:add\(new )([^(]+)(?:\([^)]*)\)\)`)

// matches Maven-style Minecraft version identifiers like
// 'com.mojang:minecraft:1.20.4' and captures the version part (e.g., "1.20.4")
var MCVER_RE = regexp.MustCompile(`(?:['"]com\.mojang:minecraft:)([0-9a-z.]+)(?:['"])`)

type repository struct {
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Stars         int    `json:"stargazers_count"`
	DefaultBranch string `json:"default_branch"`
	HtmlUrl       string `json:"html_url"`
	PushedAt      string `json:"pushed_at"`
	CreatedAt     string `json:"created_at"`
	Fork          bool   `json:"fork"`
	Archived      bool   `json:"archived"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type fabric struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Authors     []string `json:"authors"`
	Icon        string   `json:"icon"`
	Entrypoints struct {
		Meteor []string `json:"meteor"`
	} `json:"entrypoints"`
}

type release struct {
	Assets []struct {
		Name      string `json:"name"`
		Url       string `json:"browser_download_url"`
		Downloads int    `json:"download_count"`
	} `json:"assets"`
}

func getRepo(fullName string) (*repository, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v", fullName)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, "", err
	}

	var repo repository

	err = json.Unmarshal(bytes, &repo)
	if err != nil {
		return nil, "", err
	}

	return &repo, string(bytes), nil
}

func getFabricModJson(fullName string, defaultBranch string) (*fabric, string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/resources/fabric.mod.json", fullName, defaultBranch)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return nil, "", err
	}

	var fabricModJson fabric

	err = json.Unmarshal(bytes, &fabricModJson)
	if err != nil {
		return nil, "", err
	}

	return &fabricModJson, string(bytes), nil
}

// https://api.github.com/repos/{name}/releases
func getReleaseDetails(fullName string) (string, int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/releases", fullName)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", 0, err
	}

	var releases []release

	err = json.Unmarshal(bytes, &releases)
	if err != nil {
		return "", 0, err
	}

	var downloadCount int = 0
	var downloadUrl string = ""

	for _, release := range releases {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)

			if strings.HasSuffix(name, "-dev.jar") || strings.HasSuffix(name, "-sources.jar") {
				continue
			}

			if strings.HasSuffix(name, ".jar") {
				downloadCount += asset.Downloads
				if downloadUrl == "" {
					downloadUrl = asset.Url
				}
				break
			}
		}
	}

	if downloadUrl == "" {
		fmt.Println("\t\tMissing release")
	}

	return downloadUrl, downloadCount, nil
}

func getIcon(fullName string, defaultBranch string, icon string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/src/main/resources/%v", fullName, defaultBranch, icon)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	if string(bytes) == "404: Not Found" {
		fmt.Println("\t\tMissing icon")
		return "", nil
	}

	return url, nil
}

func findDiscordServer(fullName string, defaultBranch string, repoStr string, fabricStr string) (string, error) {
	// matches Discord invite links, supporting various domains
	// and formats (e.g., "https://discord.gg/abc123", "discord.com/invite/abc")
	var INVITE_RE = regexp.MustCompile(`((?:https?:\/\/)?(?:www\.)?(?:discord\.(?:gg|io|me|li|com)|discordapp\.com/invite|dsc\.gg)/[a-zA-Z0-9\-\/]+)`)

	url := fmt.Sprintf("https://raw.githubusercontent.com/%v/%v/README.md", fullName, defaultBranch)
	bytes, err := MakeGetRequest(url)
	if err != nil {
		return "", err
	}

	readme := string(bytes)

	matches := INVITE_RE.FindAllString(readme, -1)
	matches = append(matches, INVITE_RE.FindAllString(fabricStr, -1)...)
	matches = append(matches, INVITE_RE.FindAllString(repoStr, -1)...)

	for _, invite := range matches {
		if !regexp.MustCompile(`^https?://`).MatchString(invite) {
			invite = "https://" + invite
		}
		status, err := MakeHeadRequest(invite)
		if err == nil && status != 404 {
			return invite, nil
		}
	}

	return "", nil
}

func parseRepo(fullName string, number int, total int) (*Addon, error) {
	var addon Addon
	fmt.Printf("\tParsing %v, %v/%v\n", fullName, number, total)
	fmt.Printf("\t")
	SleepIfRateLimited(Core)
	repo, repoStr, err := getRepo(fullName)
	if err != nil {
		return nil, err
	}

	fabricModJson, fabricStr, err := getFabricModJson(fullName, repo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	// ensure meteor entrypoint is present
	if len(fabricModJson.Entrypoints.Meteor) == 0 {
		return nil, fmt.Errorf("Missing meteor entrypoint")
	}

	// ensure a description is present
	description := repo.Description
	if description == "" {
		description = fabricModJson.Description
	}
	if description == "" {
		fmt.Println("\t\tMissing description")
	}

	// find authors from fabric.mod.json or from github username
	var authors []string
	if len(fabricModJson.Authors) == 0 {
		authors = append(authors, repo.Owner.Login)
	} else {
		authors = fabricModJson.Authors
	}

	downloadUrl, downloadCount, err := getReleaseDetails(fullName)
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

	fmt.Printf("%v\n", invite)

	_, _, _ = downloadUrl, downloadCount, icon

	return &addon, nil
}

func ParseRepos(repos [1]string) []*Addon {
	var total int = len(repos)
	var addons []*Addon
	var attempts int = RetryAttempts

	for i, fullName := range repos {
		addon, err := parseRepo(fullName, i+1, total)

		if err != nil {
			if attempts == 0 {
				fmt.Printf("Failed to parse %v repositories -> something is very wrong", RetryAttempts)
			}
			fmt.Printf("\tFailed to parse %v: %v", fullName, err)
			attempts -= 1
			continue
		}
		addons = append(addons, addon)
	}

	return addons
}
